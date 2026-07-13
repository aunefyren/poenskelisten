package main

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/controllers"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/mcpserver"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	textTemplate "text/template"

	"strconv"
	"time"

	_ "time/tzdata"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	utilities.PrintASCII()
	gin.SetMode(gin.ReleaseMode)

	// Create files directory
	newPath := filepath.Join(".", "files")
	err := os.MkdirAll(newPath, os.ModePerm)
	if err != nil {
		fmt.Println("failed to create 'files' directory. error: " + err.Error())
		os.Exit(1)
	}
	fmt.Println("directory 'files' valid")

	// Load config file
	err = config.LoadConfig()
	if err != nil {
		fmt.Println("failed to load configuration file. error: " + err.Error())
		os.Exit(1)
	}
	fmt.Println("configuration file loaded")

	// Create and define file for logging
	logger.InitLogger(config.ConfigFile)

	logger.Log.Info("running Pønskelisten version: " + config.ConfigFile.PoenskelistenVersion)

	// change the config to respect flags
	generateInvite := false
	flagsProvided := false
	config.ConfigFile, generateInvite, flagsProvided, err = parseFlags(config.ConfigFile)
	if err != nil {
		logger.Log.Error("failed to parse input flags. error: " + err.Error())
		os.Exit(1)
	}
	logger.Log.Info("flags parsed")

	// Persist the config only when flags/ENV actually overrode something, to
	// avoid rewriting config.json on every startup.
	if flagsProvided {
		err = config.SaveConfig()
		if err != nil {
			logger.Log.Error("failed to save new config. error: " + err.Error())
			os.Exit(1)
		}
	}

	// set time zone from config if it is not empty
	loc, err := time.LoadLocation(config.ConfigFile.Timezone)
	if err != nil {
		logger.Log.Error("Failed to set time zone from config. Error: " + err.Error())
		logger.Log.Warn("removing value...")

		config.ConfigFile.Timezone = "Europe/Paris"
		err = config.SaveConfig()
		if err != nil {
			logger.Log.Error("failed to set new time zone in the config. error: " + err.Error())
			os.Exit(1)
		}

	} else {
		time.Local = loc
	}
	logger.Log.Info("timezone set")

	// Initialize Database
	logger.Log.Info("connecting to database...")

	err = database.Connect(config.ConfigFile.DBType, config.ConfigFile.Timezone, config.ConfigFile.DBUsername, config.ConfigFile.DBPassword, config.ConfigFile.DBIP, config.ConfigFile.DBPort, config.ConfigFile.DBName, config.ConfigFile.DBSSL, config.ConfigFile.DBLocation)
	if err != nil {
		logger.Log.Error("failed to connect to database. error: " + err.Error())
		os.Exit(1)
	}
	database.Migrate()
	logger.Log.Info("database connected")

	if generateInvite {
		invite, err := database.GenerateRandomInvite()
		if err != nil {
			logger.Log.Error("failed to generate random invitation code. error: " + err.Error())
			os.Exit(1)
		}
		logger.Log.Info("generated new invite code. code: " + invite)
	}

	// Initialize Router
	router := initRouter(config.ConfigFile)
	logger.Log.Info("router initialized. starting Pønskelisten at http://*:" + strconv.Itoa(config.ConfigFile.PoenskelistenPort))
	log.Fatal(router.Run(":" + strconv.Itoa(config.ConfigFile.PoenskelistenPort)))
}

func initRouter(configFile models.ConfigStruct) *gin.Engine {
	router := gin.Default()

	// API endpoint
	api := router.Group("/api")
	{
		open := api.Group("/open")
		{
			open.POST("/tokens/register", controllers.GenerateToken)

			open.POST("/users", controllers.RegisterUser)
			open.POST("/users/reset", controllers.APIResetPassword)
			open.GET("/users/reset/:resetCode", controllers.APIVerifyResetCode)
			open.POST("/users/password", controllers.APIChangePassword)
			open.POST("/users/verify/:code", controllers.VerifyUser)
			open.POST("/users/verification", controllers.SendUserVerificationCode)

			// MFA enrollment is reachable during the login gate (SSO-authed), so it
			// lives under /open: a user forced to enroll has no access token yet.
			open.POST("/users/mfa/enroll", controllers.APIEnrollMFA)
			open.POST("/users/mfa/activate", controllers.APIActivateMFA)

			open.POST("/tokens/mfa", controllers.APIValidateMFA)

			open.GET("/oidc/config", controllers.APIGetOIDCConfig)
			open.GET("/oidc/login", controllers.OIDCLogin)
			open.GET("/oidc/callback", controllers.OIDCCallback)

			open.GET("/wishlists/public/:wishlist_hash", controllers.GetPublicWishlist)
		}

		both := api.Group("/both")
		{
			both.GET("/wishes/:wish_id/image", controllers.APIGetWishImage)
		}

		auth := api.Group("/auth").Use(middlewares.Auth(false))
		{
			auth.GET("/me", controllers.APICurrentUser)
			auth.POST("/tokens/logout-all", controllers.APILogoutAll)

			auth.GET("/connected-apps", controllers.APIListConnectedApps)
			auth.DELETE("/connected-apps/:client_id", controllers.APIRevokeConnectedApp)

			auth.GET("/currency", controllers.APIGetCurrency)

			auth.POST("/groups", controllers.RegisterGroup)
			auth.DELETE("/groups/:group_id", controllers.DeleteGroup)
			auth.POST("/groups/:group_id/join", controllers.JoinGroup)
			auth.POST("/groups/:group_id/leave", controllers.RemoveSelfFromGroup)
			auth.POST("/groups/:group_id/remove", controllers.RemoveFromGroup)
			auth.GET("/groups", controllers.GetGroups)
			auth.GET("/groups/:group_id", controllers.GetGroup)
			auth.GET("/groups/:group_id/members", controllers.GetGroupMembers)
			auth.POST("/groups/:group_id", controllers.APIUpdateGroup)
			auth.POST("/groups/:group_id/add", controllers.APIAddWishlistsToGroup)

			auth.POST("/wishlists", controllers.RegisterWishlist)
			auth.GET("/wishlists", controllers.GetWishlists)
			auth.GET("/wishlists/:wishlist_id", controllers.GetWishlist)
			auth.DELETE("/wishlists/:wishlist_id", controllers.DeleteWishlist)
			auth.POST("/wishlists/:wishlist_id/join", controllers.JoinWishlist)
			auth.POST("/wishlists/:wishlist_id/collaborate", controllers.APICollaborateWishlist)
			auth.POST("/wishlists/:wishlist_id/remove", controllers.RemoveFromWishlist)
			auth.POST("/wishlists/:wishlist_id/un-collaborate", controllers.APIUnCollaborateWishlist)
			auth.POST("/wishlists/:wishlist_id", controllers.APIUpdateWishlist)

			auth.GET("/wishlists/:wishlist_id/categories", controllers.APIGetWishlistCategories)

			auth.GET("/wishes", controllers.GetWishesFromWishlist)
			auth.POST("/wishes", controllers.RegisterWish)
			auth.DELETE("/wishes/:wish_id", controllers.DeleteWish)
			auth.POST("/wishes/:wish_id/claim", controllers.RegisterWishClaim)
			auth.POST("/wishes/:wish_id/unclaim", controllers.RemoveWishClaim)
			auth.POST("/wishes/:wish_id", controllers.APIUpdateWish)
			auth.GET("/wishes/:wish_id", controllers.APIGetWish)

			auth.GET("/news", controllers.GetNews)
			auth.GET("/news/:news_id", controllers.GetNewsPost)

			auth.GET("/users/:user_id", controllers.GetUser)
			auth.GET("/users/:user_id/image", controllers.APIGetUserProfileImage)
			auth.GET("/users", controllers.GetUsers)
			auth.POST("/users/update", controllers.UpdateUser)

			auth.POST("/users/mfa/disable", controllers.APIDisableMFA)
		}

		admin := api.Group("/admin").Use(middlewares.Auth(true))
		{
			admin.POST("/currency/update", controllers.APIUpdateCurrency)
			admin.POST("/server/settings", controllers.APIUpdateServerSettings)

			admin.DELETE("/users/:user_id", controllers.APIDeleteUser)
			admin.DELETE("/users/:user_id/mfa", controllers.APIAdminDeleteUserMFA)
			admin.DELETE("/users/:user_id/sessions", controllers.APIAdminRevokeUserSessions)

			admin.GET("/oauth/clients", controllers.APIAdminListOAuthClients)
			admin.DELETE("/oauth/clients/:client_id", controllers.APIAdminRevokeOAuthClient)

			admin.POST("/invites", controllers.RegisterInvite)
			admin.GET("/invites", controllers.APIGetAllInvites)
			admin.DELETE("/invites/:invite_id", controllers.APIDeleteInvite)

			admin.POST("/news", controllers.RegisterNewsPost)
			admin.DELETE("/news/:news_id", controllers.DeleteNewsPost)
			admin.POST("/news/:news_id", controllers.APIEditNewsPost)

			admin.POST("/server/info", controllers.APIGetServerInfo)
		}

	}

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		// AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Access-Control-Allow-Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc:  func(origin string) bool { return true },
		MaxAge:           12 * time.Hour,
	}))

	// load HTML blobs
	router.LoadHTMLGlob("./web/*/*.html")

	// Static endpoint for different directories
	router.Static("/assets", "./web/assets")
	router.Static("/css", "./web/css")

	// Create template for HTML variables
	templateData := gin.H{
		"appName":        configFile.PoenskelistenName,
		"currency":       configFile.PoenskelistenCurrency,
		"appDescription": configFile.PoenskelistenDescription,
	}

	// endpoint handler building for JS
	router, err := registerTemplatedStaticFilesForDirectory(router, "/js", true, "./web/js", templateData)
	if err != nil {
		logger.Log.Error("failed to build JS paths. error: " + err.Error())
	}

	// endpoint handler building for HTML
	router, err = registerTemplatedStaticFilesForDirectory(router, "", false, "./web/html", templateData)
	if err != nil {
		logger.Log.Error("failed to build HTML paths. error: " + err.Error())
	}

	// endpoint handler building for JSON
	router, err = registerTemplatedStaticFilesForDirectory(router, "/json", true, "./web/json", templateData)
	if err != nil {
		logger.Log.Error("failed to build JSON paths. error: " + err.Error())
	}

	// endpoint handler building for TXT
	router, err = registerTemplatedStaticFilesForDirectory(router, "/txt", true, "./web/txt", templateData)
	if err != nil {
		logger.Log.Error("failed to build TXT paths. error: " + err.Error())
	}

	// OAuth 2.1 / MCP discovery documents live at the domain root, outside /api.
	router.GET("/.well-known/oauth-authorization-server", controllers.APIOAuthAuthorizationServerMetadata)
	router.GET("/.well-known/oauth-protected-resource", controllers.APIOAuthProtectedResourceMetadata)
	router.GET("/.well-known/jwks.json", controllers.APIOAuthJWKS)

	// OAuth 2.1 authorization server endpoints (also outside /api).
	router.GET("/oauth/authorize", controllers.APIOAuthAuthorize)
	router.POST("/oauth/consent", controllers.APIOAuthConsent)
	router.POST("/oauth/token", controllers.APIOAuthToken)
	router.POST("/oauth/revoke", controllers.APIOAuthRevoke)
	// Open dynamic client registration (RFC 7591), rate-limited per IP.
	router.POST("/oauth/register", middlewares.RateLimit(10, time.Hour), controllers.APIOAuthRegister)

	// MCP resource server (self-gates on MCPEnabled; OAuth-protected).
	router.Any("/mcp", mcpserver.Handler())

	return router
}

func parseFlags(configFile models.ConfigStruct) (models.ConfigStruct, bool, bool, error) {
	generateInviteBool := false

	// boolean values
	SMTPBool := "true"
	if configFile.SMTPEnabled {
		SMTPBool = "false"
	}

	// reverse the config bool
	dbSSLBool := "true"
	if !configFile.DBSSL {
		dbSSLBool = "false"
	}

	// Define flag variables with the configuration file as default values
	var port = flag.Int("port", configFile.PoenskelistenPort, "The port Pønskelisten is listening on.")
	var externalURL = flag.String("externalurl", configFile.PoenskelistenExternalURL, "The URL others would use to access Pønskelisten.")
	var timezone = flag.String("timezone", configFile.Timezone, "The timezone Pønskelisten is running in.")
	var environment = flag.String("environment", configFile.PoenskelistenEnvironment, "The environment Pønskelisten is running in. It will behave differently in 'test'.")
	var testemail = flag.String("testemail", configFile.PoenskelistenTestEmail, "The email all emails are sent to in test mode.")
	var name = flag.String("name", configFile.PoenskelistenName, "The name of the application. Replaces 'Pønskelisten'.")
	var description = flag.String("description", configFile.PoenskelistenDescription, "The description of the application.")
	var logLevel = flag.String("loglevel", configFile.PoenskelistenLogLevel, "The log level of the application. Default 'info'.")

	// DB values
	var dbPort = flag.Int("dbport", configFile.DBPort, "The port the database is listening on.")
	var dbType = flag.String("dbtype", configFile.DBType, "The type of database Pønskelisten is interacting with.")
	var dbUsername = flag.String("dbusername", configFile.DBUsername, "The username used to interact with the database.")
	var dbPassword = flag.String("dbpassword", configFile.DBPassword, "The password used to interact with the database.")
	var dbName = flag.String("dbname", configFile.DBName, "The database table used within the database.")
	var dbIP = flag.String("dbip", configFile.DBIP, "The IP address used to reach the database.")
	var dbSSL = flag.String("dbssl", dbSSLBool, "If the database connection uses SSL.")
	var dbLocation = flag.String("dblocation", configFile.DBLocation, "The database is a local file, what is the system file path.")

	// SMTP values
	var smtpDisabled = flag.String("disablesmtp", SMTPBool, "Disables user verification using e-mail.")
	var smtpHost = flag.String("smtphost", configFile.SMTPHost, "The SMTP server which sends e-mail.")
	var smtpPort = flag.Int("smtpport", configFile.SMTPPort, "The SMTP server port.")
	var smtpUsername = flag.String("smtpusername", configFile.SMTPUsername, "The username used to verify against the SMTP server.")
	var smtpPassword = flag.String("smtppassword", configFile.SMTPPassword, "The password used to verify against the SMTP server.")
	var smtpFrom = flag.String("smtpfrom", configFile.SMTPFrom, "The sender address when sending e-mail from Pønskelisten.")

	// Security values
	var mfaEnforced = flag.String("mfaenforced", strconv.FormatBool(configFile.MFAEnforced), "If all local users must enroll in multi-factor authentication.")
	var mfaRecoveryCodes = flag.String("mfarecoverycodes", strconv.FormatBool(configFile.MFARecoveryCodesEnabled), "If users are issued single-use recovery codes when enrolling in MFA.")

	// OIDC values
	var oidcEnabled = flag.String("oidcenabled", strconv.FormatBool(configFile.OIDCEnabled), "If OpenID Connect single sign-on is enabled.")
	var oidcProviderName = flag.String("oidcprovidername", configFile.OIDCProviderName, "The display name of the OIDC provider, shown on the login button.")
	var oidcIssuerURL = flag.String("oidcissuerurl", configFile.OIDCIssuerURL, "The OIDC issuer URL (used for discovery).")
	var oidcClientID = flag.String("oidcclientid", configFile.OIDCClientID, "The OIDC client ID.")
	var oidcClientSecret = flag.String("oidcclientsecret", configFile.OIDCClientSecret, "The OIDC client secret.")
	var oidcRedirectURL = flag.String("oidcredirecturl", configFile.OIDCRedirectURL, "The OIDC redirect/callback URL registered with the provider.")
	var oidcAutoCreate = flag.String("oidcautocreateusers", strconv.FormatBool(configFile.OIDCAutoCreateUsers), "If unknown OIDC users are automatically provisioned an account.")

	// MCP toggle. The OAuth issuer/algorithm and the API/MCP resource identifiers
	// auto-derive from the external URL and can be hand-edited in config.json if a
	// deployment ever needs to override them.
	var mcpEnabled = flag.String("mcpenabled", strconv.FormatBool(configFile.MCPEnabled), "If the MCP resource server is enabled.")

	// Generate invite
	var generateInvite = flag.String("generateinvite", "false", "If an invite code should be automatically generate on startup.")

	// Parse flags
	flag.Parse()

	// flag pointers are never nil after Parse(), so the only reliable way to
	// tell a deliberately-set flag from one left at its config default is to
	// record which flags were actually provided on the command line. Only
	// provided flags override the loaded configuration.
	provided := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { provided[f.Name] = true })
	flagsProvided := len(provided) > 0

	if provided["port"] {
		configFile.PoenskelistenPort = *port
	}

	if provided["externalurl"] {
		configFile.PoenskelistenExternalURL = *externalURL
	}

	if provided["timezone"] {
		configFile.Timezone = *timezone
	}

	if provided["environment"] {
		configFile.PoenskelistenEnvironment = *environment
	}

	if provided["testemail"] {
		configFile.PoenskelistenTestEmail = *testemail
	}

	if provided["description"] {
		configFile.PoenskelistenDescription = *description
	}

	if provided["name"] {
		configFile.PoenskelistenName = *name
	}

	if provided["loglevel"] {
		parsedLogLevel, parseErr := logrus.ParseLevel(*logLevel)
		if parseErr == nil {
			configFile.PoenskelistenLogLevel = parsedLogLevel.String()
			logger.Log.SetLevel(parsedLogLevel)
			logger.Log.Info("Log level changed to: " + parsedLogLevel.String())
		} else {
			logger.Log.Warn("Failed to parse log level: " + *logLevel)
		}
	}

	if provided["dbport"] {
		configFile.DBPort = *dbPort
	}

	if provided["dbtype"] {
		configFile.DBType = *dbType
	}

	if provided["dbusername"] {
		configFile.DBUsername = *dbUsername
	}

	if provided["dbpassword"] {
		configFile.DBPassword = *dbPassword
	}

	if provided["dbname"] {
		configFile.DBName = *dbName
	}

	if provided["dbip"] {
		configFile.DBIP = *dbIP
	}

	// Parsed as a string so the "--dbssl true/false" calling convention keeps working.
	if provided["dbssl"] {
		configFile.DBSSL = strings.ToLower(*dbSSL) == "true"
	}

	if provided["dblocation"] {
		configFile.DBLocation = *dbLocation
	}

	// disablesmtp=true disables e-mail; disablesmtp=false enables it. Parsed as
	// a string so the "--disablesmtp true/false" calling convention keeps working.
	if provided["disablesmtp"] {
		configFile.SMTPEnabled = strings.ToLower(*smtpDisabled) != "true"
	}

	if provided["smtphost"] {
		configFile.SMTPHost = *smtpHost
	}

	if provided["smtpport"] {
		configFile.SMTPPort = *smtpPort
	}

	if provided["smtpusername"] {
		configFile.SMTPUsername = *smtpUsername
	}

	if provided["smtppassword"] {
		configFile.SMTPPassword = *smtpPassword
	}

	if provided["smtpfrom"] {
		configFile.SMTPFrom = *smtpFrom
	}

	// Parsed as a string so the "--mfaenforced true/false" calling convention
	// matches the other boolean flags.
	if provided["mfaenforced"] {
		configFile.MFAEnforced = strings.ToLower(*mfaEnforced) == "true"
	}

	if provided["mfarecoverycodes"] {
		configFile.MFARecoveryCodesEnabled = strings.ToLower(*mfaRecoveryCodes) == "true"
	}

	if provided["oidcenabled"] {
		configFile.OIDCEnabled = strings.ToLower(*oidcEnabled) == "true"
	}

	if provided["oidcprovidername"] {
		configFile.OIDCProviderName = *oidcProviderName
	}

	if provided["oidcissuerurl"] {
		configFile.OIDCIssuerURL = *oidcIssuerURL
	}

	if provided["oidcclientid"] {
		configFile.OIDCClientID = *oidcClientID
	}

	if provided["oidcclientsecret"] {
		configFile.OIDCClientSecret = *oidcClientSecret
	}

	if provided["oidcredirecturl"] {
		configFile.OIDCRedirectURL = *oidcRedirectURL
	}

	if provided["oidcautocreateusers"] {
		configFile.OIDCAutoCreateUsers = strings.ToLower(*oidcAutoCreate) == "true"
	}

	if provided["mcpenabled"] {
		configFile.MCPEnabled = strings.ToLower(*mcpEnabled) == "true"
	}

	// Runtime-only action, never persisted to config.
	if provided["generateinvite"] {
		generateInviteBool = strings.ToLower(*generateInvite) == "true"
	}

	// Failsafe, if port is 0, set to default 8080
	if configFile.PoenskelistenPort == 0 {
		configFile.PoenskelistenPort = 8080
	}

	return configFile, generateInviteBool, flagsProvided, nil
}

func registerTemplatedStaticFilesForDirectory(
	r *gin.Engine,
	urlPrefix string,
	keepFileExtension bool,
	fileDirectory string,
	templateData any,
) (
	newRouter *gin.Engine,
	err error,
) {
	type filePathTable struct {
		urlPath string
	}
	filePathTableList := map[string]*filePathTable{}
	filePathTableList["frontpage.html"] = &filePathTable{urlPath: "/"}
	filePathTableList["group.html"] = &filePathTable{urlPath: "/groups/:group_id"}
	filePathTableList["wishlist.html"] = &filePathTable{urlPath: "/wishlists/:wishlist_id"}
	filePathTableList["public.html"] = &filePathTable{urlPath: "/wishlists/public/:wishlist_hash"}
	filePathTableList["callback.html"] = &filePathTable{urlPath: "/oauth/callback"}
	filePathTableList["enroll.html"] = &filePathTable{urlPath: "/enroll"}
	filePathTableList["manifest.json"] = &filePathTable{urlPath: "/manifest.json"}
	filePathTableList["service-worker.js"] = &filePathTable{urlPath: "/service-worker.js"}
	filePathTableList["robots.txt"] = &filePathTable{urlPath: "/robots.txt"}

	root := os.DirFS(fileDirectory)
	jsTemplates := MustLoadTemplates("./web/js/*.js")
	jsonTemplates := MustLoadTemplates("./web/json/*.json")
	txtTemplates := MustLoadTemplates("./web/txt/*.txt")

	foundFiles, err := fs.Glob(root, "*")

	if err != nil {
		logger.Log.Error("failed to load directory. error: " + err.Error())
		return
	}

	logger.Log.Debug("found " + strconv.Itoa(len(foundFiles)) + " files for endpoint mapping using path: " + fileDirectory)

	for _, file := range foundFiles {
		fileWithoutExtension := file
		extension := filepath.Ext(file)

		if !keepFileExtension && strings.Contains(file, ".") {
			fileWithoutExtension = strings.TrimSuffix(file, extension)
		}

		path := urlPrefix + "/" + fileWithoutExtension

		if filePathTableList[file] != nil {
			path = filePathTableList[file].urlPath
		}

		switch strings.ToLower(extension) {
		case ".html":
			r.GET(path, func(c *gin.Context) {
				c.HTML(http.StatusOK, file, templateData)
			})
			logger.Log.Debug("registered HTML '" + file + "' to path '" + path + "'")
		case ".js":
			r.GET(path, RenderJSTemplate(jsTemplates, file, templateData))
			logger.Log.Debug("registered JS '" + file + "' to path '" + path + "'")
		case ".json":
			r.GET(path, RenderJSONTemplate(jsonTemplates, file, templateData))
			logger.Log.Debug("registered JSON '" + file + "' to path '" + path + "'")
		case ".txt":
			r.GET(path, RenderTextTemplate(txtTemplates, file, templateData))
			logger.Log.Debug("registered TXT '" + file + "' to path '" + path + "'")
		}
	}

	return r, err
}

func MustLoadTemplates(glob string) *textTemplate.Template {
	t, err := textTemplate.ParseGlob(glob)
	if err != nil {
		logger.Log.Warn("failed to parse file: " + glob)
	}
	return t
}

func RenderJSTemplate(jsTemplates *textTemplate.Template, name string, data any) gin.HandlerFunc {
	return func(c *gin.Context) {
		var buf bytes.Buffer
		if err := jsTemplates.ExecuteTemplate(&buf, name, data); err != nil {
			c.String(http.StatusInternalServerError, "template error: %v", err)
			return
		}

		c.Data(http.StatusOK, "application/javascript; charset=utf-8", buf.Bytes())
	}
}

func RenderJSONTemplate(jsTemplates *textTemplate.Template, name string, data any) gin.HandlerFunc {
	return func(c *gin.Context) {
		var buf bytes.Buffer
		if err := jsTemplates.ExecuteTemplate(&buf, name, data); err != nil {
			c.String(http.StatusInternalServerError, "template error: %v", err)
			return
		}

		c.Data(http.StatusOK, "application/json; charset=utf-8", buf.Bytes())
	}
}

func RenderTextTemplate(jsTemplates *textTemplate.Template, name string, data any) gin.HandlerFunc {
	return func(c *gin.Context) {
		var buf bytes.Buffer
		if err := jsTemplates.ExecuteTemplate(&buf, name, data); err != nil {
			c.String(http.StatusInternalServerError, "template error: %v", err)
			return
		}

		c.Data(http.StatusOK, "text/plain; charset=utf-8", buf.Bytes())
	}
}

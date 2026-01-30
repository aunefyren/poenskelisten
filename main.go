package main

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/controllers"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
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
		fmt.Println("Failed to create 'files' directory. Error: " + err.Error())
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

	logger.Log.Info("Running Pønskelisten version: " + config.ConfigFile.PoenskelistenVersion)

	// change the config to respect flags
	generateInvite := false
	config.ConfigFile, generateInvite, err = parseFlags(config.ConfigFile)
	if err != nil {
		logger.Log.Error("Failed to parse input flags. Error: " + err.Error())
		os.Exit(1)
	}
	logger.Log.Info("flags parsed")

	// save new version of config
	err = config.SaveConfig()
	if err != nil {
		logger.Log.Error("Failed to set new time zone in the config. Error: " + err.Error())
		os.Exit(1)
	}

	// set time zone from config if it is not empty
	loc, err := time.LoadLocation(config.ConfigFile.Timezone)
	if err != nil {
		logger.Log.Error("Failed to set time zone from config. Error: " + err.Error())
		logger.Log.Warn("Removing value...")

		config.ConfigFile.Timezone = "Europe/Paris"
		err = config.SaveConfig()
		if err != nil {
			logger.Log.Error("Failed to set new time zone in the config. Error: " + err.Error())
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
		logger.Log.Error("Failed to connect to database. Error: " + err.Error())
		os.Exit(1)
	}
	database.Migrate()
	logger.Log.Info("database connected")

	if generateInvite {
		invite, err := database.GenerateRandomInvite()
		if err != nil {
			logger.Log.Error("Failed to generate random invitation code. Error: " + err.Error())
			os.Exit(1)
		}
		logger.Log.Info("Generated new invite code. Code: " + invite)
	}

	// Initialize Router
	router := initRouter(config.ConfigFile)
	logger.Log.Info("Router initialized. Starting Pønskelisten at http://*:" + strconv.Itoa(config.ConfigFile.PoenskelistenPort))
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

			open.GET("/wishlists/public/:wishlist_hash", controllers.GetPublicWishlist)
		}

		both := api.Group("/both")
		{
			both.GET("/wishes/:wish_id/image", controllers.APIGetWishImage)
		}

		auth := api.Group("/auth").Use(middlewares.Auth(false))
		{
			auth.POST("/tokens/validate", controllers.ValidateToken)

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
		}

		admin := api.Group("/admin").Use(middlewares.Auth(true))
		{
			admin.POST("/currency/update", controllers.APIUpdateCurrency)

			admin.DELETE("/users/:user_id", controllers.APIDeleteUser)

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

	return router
}

func parseFlags(configFile models.ConfigStruct) (models.ConfigStruct, bool, error) {
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
	var description = flag.String("description", configFile.PoenskelistenName, "The description of the application.")
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

	// Generate invite
	var generateInvite = flag.String("generateinvite", "false", "If an invite code should be automatically generate on startup.")

	// Parse flags
	flag.Parse()

	// Respect the flag if provided
	if port != nil {
		configFile.PoenskelistenPort = *port
	}

	// Respect the flag if provided
	if externalURL != nil {
		configFile.PoenskelistenExternalURL = *externalURL
	}

	// Respect the flag if provided
	if timezone != nil {
		configFile.Timezone = *timezone
	}

	// Respect the flag if provided
	if environment != nil {
		configFile.PoenskelistenEnvironment = *environment
	}

	// Respect the flag if provided
	if testemail != nil {
		configFile.PoenskelistenTestEmail = *testemail
	}

	// Respect the flag if provided
	if description != nil {
		configFile.PoenskelistenDescription = *description
	}

	// Respect the flag if provided
	if name != nil {
		configFile.PoenskelistenName = *name
	}

	// Respect the flag if provided
	if logLevel != nil && *logLevel != configFile.PoenskelistenLogLevel {
		parsedLogLevel, err := logrus.ParseLevel(*logLevel)
		if err == nil {
			configFile.PoenskelistenLogLevel = parsedLogLevel.String()
			logger.Log.SetLevel(parsedLogLevel)
			logger.Log.Info("Log level changed to: " + parsedLogLevel.String())
		} else {
			logger.Log.Warn("Failed to parse log level: " + *logLevel)
		}
	}

	// Respect the flag if provided
	if dbPort != nil {
		configFile.DBPort = *dbPort
	}

	// Respect the flag if provided
	if dbType != nil {
		configFile.DBType = *dbType
	}

	// Respect the flag if provided
	if dbUsername != nil {
		configFile.DBUsername = *dbUsername
	}

	// Respect the flag if provided
	if dbPassword != nil {
		configFile.DBPassword = *dbPassword
	}

	// Respect the flag if provided
	if dbName != nil {
		configFile.DBName = *dbName
	}

	// Respect the flag if provided
	if dbIP != nil {
		configFile.DBIP = *dbIP
	}

	// Respect the flag if string is true
	if dbSSL != nil {
		dbSSLBool := false
		if strings.ToLower(*dbSSL) == "true" {
			dbSSLBool = true
		}
		configFile.DBSSL = dbSSLBool
	}

	// Respect the flag if provided
	if dbLocation != nil {
		configFile.DBLocation = *dbLocation
	}

	// Respect the flag if string is true
	if smtpDisabled != nil {
		if strings.ToLower(*smtpDisabled) == "true" {
			configFile.SMTPEnabled = false
		}
	}

	// Respect the flag if provided
	if smtpHost != nil {
		configFile.SMTPHost = *smtpHost
	}

	// Respect the flag if provided
	if smtpPort != nil {
		configFile.SMTPPort = *smtpPort
	}

	// Respect the flag if provided
	if smtpUsername != nil {
		configFile.SMTPUsername = *smtpUsername
	}

	// Respect the flag if provided
	if smtpPassword != nil {
		configFile.SMTPPassword = *smtpPassword
	}

	// Respect the flag if provided
	if smtpFrom != nil {
		configFile.SMTPFrom = *smtpFrom
	}

	// Respect the flag if string is true
	if generateInvite != nil {
		if strings.ToLower(*generateInvite) == "true" {
			generateInviteBool = true
		}
	}

	// Failsafe, if port is 0, set to default 8080
	if configFile.PoenskelistenPort == 0 {
		configFile.PoenskelistenPort = 8080
	}

	return configFile, generateInviteBool, nil
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

package main

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/controllers"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	fmt.Println("Directory 'files' valid.")

	// Load config file
	configFile, err := config.GetConfig()
	if err != nil {
		fmt.Println("Failed to load configuration file. Error: " + err.Error())
		os.Exit(1)
	}
	fmt.Println("Configuration file loaded.")

	// Create and define file for logging
	logger.InitLogger(configFile)

	logger.Log.Info("Running Pønskelisten version: " + configFile.PoenskelistenVersion)

	// Change the config to respect flags
	configFile, generateInvite, err := parseFlags(configFile)
	if err != nil {
		logger.Log.Error("Failed to parse input flags. Error: " + err.Error())
		os.Exit(1)
	}
	logger.Log.Info("Flags parsed.")

	// Set time zone from config if it is not empty
	loc, err := time.LoadLocation(configFile.Timezone)
	if err != nil {
		logger.Log.Error("Failed to set time zone from config. Error: " + err.Error())
		logger.Log.Warn("Removing value...")

		configFile.Timezone = "Europe/Paris"
		err = config.SaveConfig(configFile)
		if err != nil {
			logger.Log.Error("Failed to set new time zone in the config. Error: " + err.Error())
			os.Exit(1)
		}

	} else {
		time.Local = loc
	}
	logger.Log.Info("Timezone set.")

	// Initialize Database
	logger.Log.Info("Connecting to database...")

	err = database.Connect(configFile.DBType, configFile.Timezone, configFile.DBUsername, configFile.DBPassword, configFile.DBIP, configFile.DBPort, configFile.DBName, configFile.DBSSL, configFile.DBLocation)
	if err != nil {
		logger.Log.Error("Failed to connect to database. Error: " + err.Error())
		os.Exit(1)
	}
	database.Migrate()
	logger.Log.Info("Database connected.")

	if generateInvite {
		invite, err := database.GenerateRandomInvite()
		if err != nil {
			logger.Log.Error("Failed to generate random invitation code. Error: " + err.Error())
			os.Exit(1)
		}
		logger.Log.Info("Generated new invite code. Code: " + invite)
	}

	// Initialize Router
	router := initRouter()
	logger.Log.Info("Router initialized. Starting Pønskelisten at http://*:" + strconv.Itoa(configFile.PoenskelistenPort))
	log.Fatal(router.Run(":" + strconv.Itoa(configFile.PoenskelistenPort)))
}

func initRouter() *gin.Engine {
	router := gin.Default()

	router.LoadHTMLGlob("web/*/*.html")

	// API endpoint
	api := router.Group("/api")
	{
		open := api.Group("/open")
		{
			open.POST("/tokens/register", controllers.GenerateToken)

			open.POST("/users", controllers.RegisterUser)
			open.POST("/users/reset", controllers.APIResetPassword)
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

	// Static endpoint for different directories
	router.Static("/assets", "./web/assets")
	router.Static("/css", "./web/css")
	router.Static("/js", "./web/js")
	router.Static("/json", "./web/json")

	// Static endpoint for homepage
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "frontpage.html", nil)
	})

	// Static endpoint for selecting your group
	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	// Static endpoint for selecting your group
	router.GET("/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register.html", nil)
	})

	// Static endpoint for your own account
	router.GET("/account", func(c *gin.Context) {
		c.HTML(http.StatusOK, "account.html", nil)
	})

	// Static endpoint for selecting your group
	router.GET("/groups", func(c *gin.Context) {
		c.HTML(http.StatusOK, "groups.html", nil)
	})

	// Static endpoint for details in your group
	router.GET("/groups/:group_id", func(c *gin.Context) {
		c.HTML(http.StatusOK, "group.html", nil)
	})

	// Static endpoint for wishlist in your group
	router.GET("/wishlists", func(c *gin.Context) {
		c.HTML(http.StatusOK, "wishlists.html", nil)
	})

	// Static endpoint for wishlist in your group
	router.GET("/wishlists/:wishlist_id", func(c *gin.Context) {
		c.HTML(http.StatusOK, "wishlist.html", nil)
	})

	// Static endpoint for public wishlist
	router.GET("/wishlists/public/:wishlist_hash", func(c *gin.Context) {
		c.HTML(http.StatusOK, "public.html", nil)
	})

	// Static endpoint for admin panel
	router.GET("/admin", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin.html", nil)
	})

	// Static endpoint for verifying account
	router.GET("/verify", func(c *gin.Context) {
		c.HTML(http.StatusOK, "verify.html", nil)
	})

	// Static endpoint for service-worker
	router.GET("/service-worker.js", func(c *gin.Context) {
		JSfile, err := os.ReadFile("./web/js/service-worker.js")
		if err != nil {
			logger.Log.Error("Reading service-worker threw error trying to open the file. Error: " + err.Error())
		}
		c.Data(http.StatusOK, "text/javascript", JSfile)
	})

	// Static endpoint for manifest
	router.GET("/manifest.json", func(c *gin.Context) {
		JSONfile, err := os.ReadFile("./web/json/manifest.json")
		if err != nil {
			logger.Log.Error("Reading manifest threw error trying to open the file. Error: " + err.Error())
		}
		c.Data(http.StatusOK, "text/json", JSONfile)
	})

	// Static endpoint for robots.txt
	router.GET("/robots.txt", func(c *gin.Context) {
		TXTfile, err := os.ReadFile("./web/txt/robots.txt")
		if err != nil {
			logger.Log.Error("Reading manifest threw error trying to open the file. Error: " + err.Error())
		}
		c.Data(http.StatusOK, "text/plain", TXTfile)
	})

	return router
}

func parseFlags(configFile models.ConfigStruct) (models.ConfigStruct, bool, error) {
	generateInviteBool := false

	// boolean values
	SMTPBool := "true"
	if !configFile.SMTPEnabled {
		SMTPBool = "false"
	}
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

	// Save the new configFile
	err := config.SaveConfig(configFile)
	if err != nil {
		return models.ConfigStruct{}, false, err
	}

	return configFile, generateInviteBool, nil
}

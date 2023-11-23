package main

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/controllers"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"flag"
	"fmt"
	"io"
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
	"github.com/thanhpk/randstr"
)

func main() {
	utilities.PrintASCII()
	gin.SetMode(gin.ReleaseMode)

	// Create files directory
	newpath := filepath.Join(".", "files")
	err := os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		fmt.Println("Failed to create 'files' directory. Error: " + err.Error())

		os.Exit(1)
	}
	fmt.Println("Directory 'files' valid.")

	// Create and define file for logging
	logFile, err := os.OpenFile("files/poenskelisten.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("Failed to load log file. Error: " + err.Error())

		os.Exit(1)
	}

	// Set log file as log destination
	log.SetOutput(logFile)
	fmt.Println("Log file set.")

	var mw io.Writer

	out := os.Stdout
	mw = io.MultiWriter(out, logFile)

	// Get pipe reader and writer | writes to pipe writer come out pipe reader
	_, w, _ := os.Pipe()

	// Replace stdout,stderr with pipe writer | all writes to stdout, stderr will go through pipe instead (log.print, log)
	os.Stdout = w
	os.Stderr = w

	// writes with log.Print should also write to mw
	log.SetOutput(mw)

	// Load config file
	Config, err := config.GetConfig()
	if err != nil {
		log.Println("Failed to load configuration file. Error: " + err.Error())

		os.Exit(1)
	}
	log.Println("Configuration file loaded.")

	// Change the config to respect flags
	Config, generateInvite, upgradeToV2, err := parseFlags(Config)
	if err != nil {
		log.Println("Failed to parse input flags. Error: " + err.Error())

		os.Exit(1)
	}
	log.Println("Flags parsed.")

	if upgradeToV2 {
		utilities.MigrateDBToV2()
		os.Exit(1)
	}

	// Set time zone from config if it is not empty
	if Config.Timezone != "" {
		loc, err := time.LoadLocation(Config.Timezone)
		if err != nil {
			log.Println("Failed to set time zone from config. Error: " + err.Error())
			log.Println("Removing value...")

			Config.Timezone = ""
			err = config.SaveConfig(Config)
			if err != nil {
				log.Println("Failed to set new time zone in the config. Error: " + err.Error())

				os.Exit(1)
			}

		} else {
			time.Local = loc
		}
	}
	log.Println("Timezone set.")

	if Config.PrivateKey == "" || len(Config.PrivateKey) < 16 {
		log.Println("Creating new private key.")

		Config.PrivateKey = randstr.Hex(32)
		config.SaveConfig(Config)
	}

	err = auth.SetPrivateKey(Config.PrivateKey)
	if Config.PrivateKey == "" || len(Config.PrivateKey) < 16 {
		log.Println("Failed to set private key. Error: " + err.Error())

		os.Exit(1)
	}
	log.Println("Private key set.")

	// Initialize Database
	log.Println("Connecting to database...")

	err = database.Connect(Config.DBType, Config.Timezone, Config.DBUsername, Config.DBPassword, Config.DBIP, Config.DBPort, Config.DBName, Config.DBSSL, Config.DBLocation)
	if err != nil {
		log.Println("Failed to connect to database. Error: " + err.Error())
		os.Exit(1)
	}
	database.Migrate()

	log.Println("Database connected.")

	if generateInvite {
		invite, err := database.GenrateRandomInvite()
		if err != nil {
			log.Println("Failed to generate random invitation code. Error: " + err.Error())

			os.Exit(1)
		}
		log.Println("Generated new invite code. Code: " + invite)
	}

	// Initialize Router
	router := initRouter()

	log.Println("Router initialized.")

	log.Fatal(router.Run(":" + strconv.Itoa(Config.PoenskelistenPort)))
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
			auth.POST("/groups/:group_id/update", controllers.APIUpdateGroup)

			auth.POST("/wishlists", controllers.RegisterWishlist)
			auth.GET("/wishlists", controllers.GetWishlists)
			auth.GET("/wishlists/:wishlist_id", controllers.GetWishlist)
			auth.DELETE("/wishlists/:wishlist_id", controllers.DeleteWishlist)
			auth.POST("/wishlists/:wishlist_id/join", controllers.JoinWishlist)
			auth.POST("/wishlists/:wishlist_id/collaborate", controllers.APICollaborateWishlist)
			auth.POST("/wishlists/:wishlist_id/remove", controllers.RemoveFromWishlist)
			auth.POST("/wishlists/:wishlist_id/un-collaborate", controllers.APIUnCollaborateWishlist)
			auth.POST("/wishlists/:wishlist_id/update", controllers.APIUpdateWishlist)

			auth.GET("/wishes", controllers.GetWishesFromWishlist)
			auth.POST("/wishes", controllers.RegisterWish)
			auth.DELETE("/wishes/:wish_id", controllers.DeleteWish)
			auth.POST("/wishes/:wish_id/claim", controllers.RegisterWishClaim)
			auth.POST("/wishes/:wish_id/unclaim", controllers.RemoveWishClaim)
			auth.POST("/wishes/:wish_id/update", controllers.APIUpdateWish)
			auth.GET("/wishes/:wish_id/image", controllers.APIGetWishImage)
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

			admin.POST("/invites", controllers.RegisterInvite)
			admin.GET("/invites", controllers.APIGetAllInvites)
			admin.DELETE("/invites/:invite_id", controllers.APIDeleteInvite)

			admin.POST("/news", controllers.RegisterNewsPost)
			admin.DELETE("/news/:news_id", controllers.DeleteNewsPost)

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
			fmt.Println("Reading service-worker threw error trying to open the file. Error: " + err.Error())
		}
		c.Data(http.StatusOK, "text/javascript", JSfile)
	})

	// Static endpoint for manifest
	router.GET("/manifest.json", func(c *gin.Context) {
		JSONfile, err := os.ReadFile("./web/manifest.json")
		if err != nil {
			fmt.Println("Reading manifest threw error trying to open the file. Error: " + err.Error())
		}
		c.Data(http.StatusOK, "text/json", JSONfile)
	})

	// Static endpoint for robots.txt
	router.GET("/robots.txt", func(c *gin.Context) {
		TXTfile, err := os.ReadFile("./web/txt/robots.txt")
		if err != nil {
			fmt.Println("Reading manifest threw error trying to open the file. Error: " + err.Error())
		}
		c.Data(http.StatusOK, "text/plain", TXTfile)
	})

	return router
}

func parseFlags(Config *models.ConfigStruct) (*models.ConfigStruct, bool, bool, error) {

	// Define flag variables with the configuration file as default values
	var port int
	flag.IntVar(&port, "port", Config.PoenskelistenPort, "The port Pønskelisten is listening on.")

	var externalURL string
	flag.StringVar(&externalURL, "externalurl", Config.PoenskelistenExternalURL, "The URL others would use to access Pønskelisten.")

	var timezone string
	flag.StringVar(&timezone, "timezone", Config.Timezone, "The timezone Pønskelisten is running in.")

	var dbPort int
	flag.IntVar(&dbPort, "dbport", Config.DBPort, "The port the database is listening on.")

	var dbType string
	flag.StringVar(&dbType, "dbtype", Config.DBType, "The type of database Pønskelisten is interacting with.")

	var dbUsername string
	flag.StringVar(&dbUsername, "dbusername", Config.DBUsername, "The username used to interact with the database.")

	var dbPassword string
	flag.StringVar(&dbPassword, "dbpassword", Config.DBPassword, "The password used to interact with the database.")

	var dbName string
	flag.StringVar(&dbName, "dbname", Config.DBName, "The database table used within the database.")

	var dbIP string
	flag.StringVar(&dbIP, "dbip", Config.DBIP, "The IP address used to reach the database.")

	var dbSSL string
	var dbSSLBool bool
	flag.StringVar(&dbSSL, "dbssl", "false", "If the database connection uses SSL.")

	var dbLocation string
	flag.StringVar(&dbLocation, "dblocation", "", "The database is a local file, what is the system file path.")

	var smtpDisabled string
	flag.StringVar(&smtpDisabled, "disablesmtp", "false", "Disables user verification using e-mail.")

	var smtpHost string
	flag.StringVar(&smtpHost, "smtphost", Config.SMTPHost, "The SMTP server which sends e-mail.")

	var smtpPort int
	flag.IntVar(&smtpPort, "smtpport", Config.SMTPPort, "The SMTP server port.")

	var smtpUsername string
	flag.StringVar(&smtpUsername, "smtpusername", Config.SMTPUsername, "The username used to verify against the SMTP server.")

	var smtpPassword string
	flag.StringVar(&smtpPassword, "smtppassword", Config.SMTPPassword, "The password used to verify against the SMTP server.")

	var smtpFrom string
	flag.StringVar(&smtpFrom, "smtpfrom", Config.SMTPFrom, "The sender address when sending e-mail from Pønskelisten.")

	var generateInvite string
	var generateInviteBool bool
	flag.StringVar(&generateInvite, "generateinvite", "false", "If an invite code should be automatically generate on startup.")

	var upgradeToV2 string
	var upgradeToV2Bool bool
	flag.StringVar(&upgradeToV2, "upgradetov2", "false", "If have placed your old pre-V2 database .json in the files folder as 'db.json' we will attempt to migrate the data.")

	// Parse the flags from input
	flag.Parse()

	// Respect the flag if config is empty
	if Config.PoenskelistenPort == 0 {
		Config.PoenskelistenPort = port
	}

	// Respect the flag if config is empty
	if Config.PoenskelistenExternalURL == "" {
		Config.PoenskelistenExternalURL = externalURL
	}

	// Respect the flag if config is empty
	if Config.Timezone == "" {
		Config.Timezone = timezone
	}

	// Respect the flag if config is empty
	if Config.DBPort == 0 {
		Config.DBPort = dbPort
	}

	// Respect the flag if config is empty
	if Config.DBType == "" {
		Config.DBType = dbUsername
	}

	// Respect the flag if config is empty
	if Config.DBUsername == "" {
		Config.DBUsername = dbUsername
	}

	// Respect the flag if config is empty
	if Config.DBPassword == "" {
		Config.DBPassword = dbPassword
	}

	// Respect the flag if config is empty
	if Config.DBName == "" {
		Config.DBName = dbName
	}

	// Respect the flag if config is empty
	if Config.DBIP == "" {
		Config.DBIP = dbIP
	}

	// Respect the flag if string is true
	if strings.ToLower(dbSSL) == "true" {
		dbSSLBool = true
	} else {
		dbSSLBool = false
	}
	Config.DBSSL = dbSSLBool

	// Respect the flag if config is empty
	if Config.DBLocation == "" {
		Config.DBLocation = dbLocation
	}

	// Respect the flag if string is true
	if strings.ToLower(smtpDisabled) == "true" {
		Config.SMTPEnabled = false
	}

	// Respect the flag if config is empty
	if Config.SMTPHost == "" {
		Config.SMTPHost = smtpHost
	}

	// Respect the flag if config is empty
	if Config.SMTPPort == 0 {
		Config.SMTPPort = smtpPort
	}

	// Respect the flag if config is empty
	if Config.SMTPUsername == "" {
		Config.SMTPUsername = smtpUsername
	}

	// Respect the flag if config is empty
	if Config.SMTPPassword == "" {
		Config.SMTPPassword = smtpPassword
	}

	// Respect the flag if config is empty
	if Config.SMTPFrom == "" {
		Config.SMTPFrom = smtpFrom
	}

	// Respect the flag if string is true
	if strings.ToLower(generateInvite) == "true" {
		generateInviteBool = true
	} else {
		generateInviteBool = false
	}

	// Respect the flag if string is true
	if strings.ToLower(upgradeToV2) == "true" {
		upgradeToV2Bool = true
	} else {
		upgradeToV2Bool = false
	}

	// Failsafe, if port is 0, set to default 8080
	if Config.PoenskelistenPort == 0 {
		Config.PoenskelistenPort = 8080
	}

	// Save the new config
	err := config.SaveConfig(Config)
	if err != nil {
		return &models.ConfigStruct{}, false, false, err
	}

	return Config, generateInviteBool, upgradeToV2Bool, nil

}

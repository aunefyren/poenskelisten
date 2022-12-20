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

	// Create files directory
	newpath := filepath.Join(".", "files")
	err := os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		fmt.Println("Failed to create 'files' directory. Error: " + err.Error())

		os.Exit(1)
	}
	fmt.Println("Directory 'files' valid.")

	// Create and define file for logging
	Log, err := os.OpenFile("files/poenskelisten.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("Failed to load log file. Error: " + err.Error())

		os.Exit(1)
	}

	// Set log file as log destination
	log.SetOutput(Log)
	log.Println("Log file set.")
	fmt.Println("Log file set.")

	// Load config file
	Config, err := config.GetConfig()
	if err != nil {
		log.Println("Failed to load configuration file. Error: " + err.Error())
		fmt.Println("Failed to load configuration file. Error: " + err.Error())

		os.Exit(1)
	}
	log.Println("Configuration file loaded.")
	fmt.Println("Configuration file loaded.")

	// Change the config to respect flags
	Config, generateInvite, err := parseFlags(Config)
	if err != nil {
		log.Println("Failed to parse input flags. Error: " + err.Error())
		fmt.Println("Failed to parse input flags. Error: " + err.Error())

		os.Exit(1)
	}
	log.Println("Flags parsed.")
	fmt.Println("Flags parsed.")

	// Set time zone from config if it is not empty
	if Config.Timezone != "" {
		loc, err := time.LoadLocation(Config.Timezone)
		if err != nil {
			fmt.Println("Failed to set time zone from config. Error: " + err.Error())
			fmt.Println("Removing value...")

			log.Println("Failed to set time zone from config. Error: " + err.Error())
			log.Println("Removing value...")

			Config.Timezone = ""
			err = config.SaveConfig(Config)
			if err != nil {
				fmt.Println("Failed to set new time zone in the config. Error: " + err.Error())
				log.Println("Failed to set new time zone in the config. Error: " + err.Error())

				os.Exit(1)
			}

		} else {
			time.Local = loc
		}
	}
	log.Println("Timezone set.")
	fmt.Println("Timezone set.")

	if Config.PrivateKey == "" || len(Config.PrivateKey) < 16 {
		fmt.Println("Creating new private key.")
		log.Println("Creating new private key.")

		Config.PrivateKey = randstr.Hex(32)
		config.SaveConfig(Config)
	}

	err = auth.SetPrivateKey(Config.PrivateKey)
	if Config.PrivateKey == "" || len(Config.PrivateKey) < 16 {
		fmt.Println("Failed to set private key. Error: " + err.Error())
		log.Println("Failed to set private key. Error: " + err.Error())

		os.Exit(1)
	}
	log.Println("Private key set.")
	fmt.Println("Private key set.")

	// Initialize Database
	fmt.Println("Connecting to database...")
	log.Println("Connecting to database...")

	err = database.Connect(Config.DBUsername, Config.DBPassword, Config.DBIP, Config.DBPort, Config.DBName)
	if err != nil {
		fmt.Println("Failed to connect to database. Error: " + err.Error())
		log.Println("Failed to connect to database. Error: " + err.Error())

		os.Exit(1)
	}
	database.Migrate()

	log.Println("Database connected.")
	fmt.Println("Database connected.")

	if generateInvite {
		invite, err := database.GenrateRandomInvite()
		if err != nil {
			fmt.Println("Failed to generate random invitation code. Error: " + err.Error())
			log.Println("Failed to generate random invitation code. Error: " + err.Error())

			os.Exit(1)
		}
		fmt.Println("Generated new invite code. Code: " + invite)
		log.Println("Generated new invite code. Code: " + invite)
	}

	// Initialize Router
	router := initRouter()

	log.Println("Router initialized.")
	fmt.Println("Router initialized.")

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
			open.POST("/token/register", controllers.GenerateToken)
			open.POST("/user/register", controllers.RegisterUser)
			open.POST("/user/verify", controllers.VerifyUser)
		}

		auth := api.Group("/auth").Use(middlewares.Auth(false))
		{
			auth.POST("/token/validate", controllers.ValidateToken)

			auth.POST("/group/register", controllers.RegisterGroup)
			auth.POST("/group/:group_id/delete", controllers.DeleteGroup)
			auth.POST("/group/:group_id/join", controllers.JoinGroup)
			auth.POST("/group/:group_id/leave", controllers.RemoveSelfFromGroup)
			auth.POST("/group/:group_id/remove", controllers.RemoveFromGroup)
			auth.POST("/group/get", controllers.GetGroups)
			auth.POST("/group/get/:group_id", controllers.GetGroup)
			auth.POST("/group/get/:group_id/members", controllers.GetGroupMembers)

			auth.POST("/wishlist/register", controllers.RegisterWishlist)
			auth.POST("/wishlist/get", controllers.GetWishlists)
			auth.POST("/wishlist/get/:wishlist_id", controllers.GetWishlist)
			auth.POST("/wishlist/get/group/:group_id", controllers.GetWishlistsFromGroup)
			auth.POST("/wishlist/:wishlist_id/delete", controllers.DeleteWishlist)
			auth.POST("/wishlist/:wishlist_id/join", controllers.JoinWishlist)
			auth.POST("/wishlist/:wishlist_id/remove", controllers.RemoveFromWishlist)

			auth.POST("/wish/get/:wishlist_id", controllers.GetWishesFromWishlist)
			auth.POST("/wish/register/:wishlist_id", controllers.RegisterWish)
			auth.POST("/wish/:wish_id/delete", controllers.DeleteWish)
			auth.POST("/wish/:wish_id/claim", controllers.RegisterWishClaim)
			auth.POST("/wish/:wish_id/unclaim", controllers.RemoveWishClaim)

			auth.POST("/news/get", controllers.GetNews)
			auth.POST("/news/get/:news_id", controllers.GetNewsPost)

			auth.POST("/user/get/:user_id", controllers.GetUser)
			auth.POST("/user/get", controllers.GetUsers)
		}

		admin := api.Group("/admin").Use(middlewares.Auth(true))
		{
			admin.POST("/invite/register", controllers.RegisterInvite)
			admin.POST("/news/register", controllers.RegisterNewsPost)
			admin.POST("/news/:news_id/delete", controllers.DeleteNewsPost)
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

	return router
}

func parseFlags(Config *models.ConfigStruct) (*models.ConfigStruct, bool, error) {

	// Define flag variables with the configuration file as default values
	var port int
	flag.IntVar(&port, "port", Config.PoenskelistenPort, "The port Pønskelisten is listening on.")

	var timezone string
	flag.StringVar(&timezone, "timezone", Config.Timezone, "The timezone Pønskelisten is running in.")

	var dbPort int
	flag.IntVar(&dbPort, "dbport", Config.DBPort, "The port the database is listening on.")

	var dbUsername string
	flag.StringVar(&dbUsername, "dbusername", Config.DBUsername, "The username used to interact with the database.")

	var dbPassword string
	flag.StringVar(&dbPassword, "dbpassword", Config.DBPassword, "The password used to interact with the database.")

	var dbName string
	flag.StringVar(&dbName, "dbname", Config.DBName, "The database table used within the database.")

	var dbIP string
	flag.StringVar(&dbIP, "dbip", Config.DBIP, "The IP address used to reach the database.")

	var smtpDisabled string
	flag.StringVar(&smtpDisabled, "disableSMTP", "false", "Disables user verification using e-mail.")

	var smtpHost string
	flag.StringVar(&smtpHost, "smtpHost", Config.SMTPHost, "The SMTP server which sends e-mail.")

	var smtpPort int
	flag.IntVar(&smtpPort, "smtpPort", Config.SMTPPort, "The SMTP server port.")

	var smtpUsername string
	flag.StringVar(&smtpUsername, "smtpUsername", Config.SMTPUsername, "The username used to verify against the SMTP server.")

	var smtpPassword string
	flag.StringVar(&smtpPassword, "smtpPassword", Config.SMTPPassword, "The password used to verify against the SMTP server.")

	var smtpFrom string
	flag.StringVar(&smtpFrom, "smtpFrom", Config.SMTPFrom, "The sender address when sending e-mail from Pønskelisten.")

	var generateInvite string
	var generateInviteBool bool
	flag.StringVar(&generateInvite, "generateinvite", "false", "If an invite code should be automatically generate on startup.")

	// Parse the flags from input
	flag.Parse()

	// Respect the flag if config is empty
	if Config.PoenskelistenPort == 0 {
		Config.PoenskelistenPort = port
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

	// Failsafe, if port is 0, set to default 8080
	if Config.PoenskelistenPort == 0 {
		Config.PoenskelistenPort = 8080
	}

	// Save the new config
	err := config.SaveConfig(Config)
	if err != nil {
		return &models.ConfigStruct{}, false, err
	}

	return Config, generateInviteBool, nil

}

package main

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/controllers"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/utilities"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"strconv"
	"time"

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

	// Set time zone from config if it is not empty
	if Config.Timezone != "" {
		loc, err := time.LoadLocation(Config.Timezone)
		if err != nil {
			fmt.Println("Failed to set time zone from config. Error: " + err.Error())
			fmt.Println(err)
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

	/*
		fmt.Println("Sending e-mail...")
		toList := []string{"oystein.sverre@gmail.com"}
		auth := smtp.PlainAuth(Config.SMTPFrom, Config.SMTPUsername, Config.SMTPPassword, Config.SMTPHost)
		msg := "Hello geeks!!!"
		body := []byte(msg)
		smt_port_int := strconv.Itoa(Config.SMTPPort)
		err = smtp.SendMail(Config.SMTPHost+":"+smt_port_int, auth, Config.SMTPFrom, toList, body)
		if err != nil {
			fmt.Println(err)
		}
	*/

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
		}

		auth := api.Group("/auth").Use(middlewares.Auth(false))
		{
			auth.GET("/ping", controllers.Ping)

			auth.POST("/token/validate", controllers.ValidateToken)

			auth.POST("/group/register", controllers.RegisterGroup)
			auth.POST("/group/:group_id/delete", controllers.DeleteGroup)
			auth.POST("/group/:group_id/join", controllers.JoinGroup)
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

			auth.POST("/user/get/:user_id", controllers.GetUser)
			auth.POST("/user/get", controllers.GetUsers)
		}

		admin := api.Group("/admin").Use(middlewares.Auth(true))
		{
			admin.POST("/invite/register", controllers.RegisterInvite)
		}
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"https://ponskelisten.no", "https://www.ponskelisten.no", "*"},
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

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"poenskelisten/config"
	"poenskelisten/controllers"
	"poenskelisten/database"
	"poenskelisten/middlewares"
	"poenskelisten/poeutilities"

	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {

	poeutilities.PrintASCII()

	// Create files directory
	newpath := filepath.Join(".", "files")
	err := os.MkdirAll(newpath, os.ModePerm)

	// Create and define file for logging
	Log, err := os.OpenFile("files/poenskelisten.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Println("Failed to load log file. Error: ")
		log.Println(err)

		fmt.Println("Failed to load log file. Error: ")
		fmt.Println(err)

		os.Exit(1)
	}

	// Set log file as log destination
	log.SetOutput(Log)

	// Load config file
	Config, err := config.GetConfig()
	if err != nil {
		log.Println("Failed to load configuration file. Error: ")
		log.Println(err)

		fmt.Println("Failed to load configuration file. Error: ")
		fmt.Println(err)

		os.Exit(1)
	}

	// Set time zone from config if it is not empty
	if Config.Timezone != "" {
		loc, err := time.LoadLocation(Config.Timezone)
		if err != nil {
			fmt.Println("Failed to set time zone from config. Error: ")
			fmt.Println(err)
			fmt.Println("Removing value...")

			log.Println("Failed to set time zone from config. Error: ")
			log.Println(err)
			log.Println("Removing value...")

			Config.Timezone = ""
			err = config.SaveConfig(Config)
			if err != nil {
				log.Println("Failed to set new time zone in the config. Error: ")
				log.Println(err)
				log.Println("Exiting...")
				os.Exit(1)
			}

		} else {
			time.Local = loc
		}
	}

	// Initialize Database
	fmt.Println("Connecting to database...")
	log.Println("Connecting to database...")
	database.Connect(Config.DBUsername + ":" + Config.DBPassword + "@tcp(" + Config.DBIP + ":" + strconv.Itoa(Config.DBPort) + ")/" + Config.DBName + "?parseTime=true")
	database.Migrate()

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
			auth.POST("/group/:group_id/join", controllers.JoinGroup)
			auth.POST("/group/get", controllers.GetGroups)
			auth.POST("/group/get/:group_id", controllers.GetGroup)
			auth.POST("/group/get/:group_id/members", controllers.GetGroupMembers)

			auth.POST("/wishlist/register", controllers.RegisterWishlist)
			auth.POST("/wishlist/get", controllers.GetWishlists)
			auth.POST("/wishlist/get/:wishlist_id", controllers.GetWishlist)
			auth.POST("/wishlist/get/group/:group_id", controllers.GetWishlistsFromGroup)

			auth.POST("/wish/get/:group_id/:wishlist_id", controllers.GetWishesFromWishlist)
			auth.POST("/wish/register/:wishlist_id", controllers.RegisterWish)

			auth.POST("/user/get/:user_id", controllers.GetUser)
			auth.POST("/user/get", controllers.GetUsers)
		}

		admin := api.Group("/admin").Use(middlewares.Auth(true))
		{
			admin.POST("/invite/register", controllers.RegisterInvite)
		}
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:8081", "http://localhost:8080"},
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
	router.GET("/groups/:group_id/:wishlist_id", func(c *gin.Context) {
		c.HTML(http.StatusOK, "wishlist.html", nil)
	})

	return router
}

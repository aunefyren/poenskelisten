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
	"poenskelisten/util"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {

	util.PrintASCII()

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
	database.Connect(Config.DBUsername + ":" + Config.DBPassword + "@tcp(" + Config.DBIP + ":" + strconv.Itoa(Config.DBPort) + ")/" + Config.DBName + "?parseTime=true")
	database.Migrate()

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
			open.POST("/token", controllers.GenerateToken)
			open.POST("/user/register", controllers.RegisterUser)
		}

		auth := api.Group("/auth").Use(middlewares.Auth(false))
		{
			auth.GET("/ping", controllers.Ping)
		}

		admin := api.Group("/admin").Use(middlewares.Auth(true))
		{
			admin.POST("/invite/register", controllers.RegisterInvite)
		}
	}

	// Static endpoint for homepage
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "test.html", nil)
	})

	// Static endpoint for selecting your group
	router.GET("/groups/", func(c *gin.Context) {
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

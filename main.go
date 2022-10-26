package main

import (
	"fmt"
	"log"
	"os"
	"poenskelisten/config"
	"poenskelisten/controllers"
	"poenskelisten/database"
	"poenskelisten/middlewares"
	"poenskelisten/util"
	"strconv"
	"path/filepath"
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
	log.Fatal(router.Run(":8080"))

}

func initRouter() *gin.Engine {
	router := gin.Default()
	api := router.Group("/api")
	{
		api.POST("/token", controllers.GenerateToken)
		api.POST("/user/register", controllers.RegisterUser)
		secured := api.Group("/secured").Use(middlewares.Auth())
		{
			secured.GET("/ping", controllers.Ping)
		}
	}
	return router
}

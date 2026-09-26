// main.go

package main

import (
	"log"
	"os"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadEnv()

	conn, err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}

	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
			"https://cardex-nextjs.vercel.app",
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
		},
		AllowCredentials: true,
	}))

	routes.SetupRoutes(router, conn)

	port := os.Getenv("APP_PORT")

	if port == "" {
		port = "8080"
	}

	log.Println("Server running on port " + port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

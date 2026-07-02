// main.go

package main

import (
	"log"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadEnv()

	conn, err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}

	router := gin.Default()

	routes.SetupRoutes(router, conn)

	log.Println("Server running on port 8080")
	router.Run(":8080")
}

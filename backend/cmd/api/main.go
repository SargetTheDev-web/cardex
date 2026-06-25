package main

import (
	"context"
	"log"

	"backend/internal/db"
	"backend/internal/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("MAIN FILE IS RUNNING")
	conn, err := db.Connect()
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	defer conn.Close(context.Background())

	router := gin.Default()
	routes.SetupRoutes(router, conn)

	log.Println("Server running on port 8080")
	router.Run(":8080")
}

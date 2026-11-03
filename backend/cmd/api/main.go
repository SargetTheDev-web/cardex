// cmd/api/main.go

package main

import (
	"context"
	"log"
	"os"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/routes"
	"backend/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

func main() {
	config.LoadEnv()

	conn, err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}

	// Create one service instance for the overdue job and email worker.
	notificationService := service.NewNotificationService(conn, nil)

	// The worker and scheduler stop when main exits.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go notificationService.RunEmailOutboxWorker(ctx)

	// Run once at startup.
	if err := notificationService.ProcessOverdueLoans(); err != nil {
		log.Printf("initial overdue notification run failed: %v", err)
	}

	// Then check for overdue loans every hour.
	scheduler := cron.New()
	_, err = scheduler.AddFunc("@every 1h", func() {
		if err := notificationService.ProcessOverdueLoans(); err != nil {
			log.Printf("overdue notification job failed: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("register overdue notification job: %v", err)
	}

	scheduler.Start()
	defer scheduler.Stop()

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

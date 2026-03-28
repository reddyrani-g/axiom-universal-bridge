package main

import (
	"axiom-bridge/config"
	"axiom-bridge/engine"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

var appConfig = config.Load()

func init() {
	// Initialize cache
	engine.InitCache()

	// Initialize GCS (graceful fallback if not configured)
	if err := engine.InitGCS(); err != nil {
		log.Printf("Warning: GCS initialization failed: %v (will skip file uploads)", err)
	}

	// Initialize BigQuery (graceful fallback if not configured)
	if err := engine.InitBigQuery(); err != nil {
		log.Printf("Warning: BigQuery initialization failed: %v (will skip saving results)", err)
	}
}

func main() {
	log.Printf(
		"Backend configuration: port=%s vertex_mode=%s project_set=%t",
		appConfig.Port,
		engineMode(),
		appConfig.GoogleCloudProject != "",
	)

	// Setup Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Health check endpoint
	r.GET("/health", engine.HealthCheck)

	// Main processing endpoint (supports both sync and SSE)
	r.POST("/process", engine.ProcessInput)

	// Serve static frontend from public directory
	r.NoRoute(gin.WrapH(http.FileServer(http.Dir("./public"))))

	port := appConfig.Port

	log.Printf("🌉 Starting Axiom Universal Bridge on port %s", port)

	// Create server with graceful shutdown
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Handle graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("Shutting down gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		// Cleanup cloud clients
		if err := engine.CloseGCS(); err != nil {
			log.Printf("GCS close error: %v", err)
		}
		if err := engine.CloseBigQuery(); err != nil {
			log.Printf("BigQuery close error: %v", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func engineMode() string {
	if appConfig.AIMode == config.AIModeMock || appConfig.AIMode == config.AIModeLocal || appConfig.GoogleCloudProject == "" {
		return "mock"
	}
	return "vertex"
}

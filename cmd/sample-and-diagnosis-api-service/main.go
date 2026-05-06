package main

import (
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lukasmichalcak/sample-and-diagnosis-webapi/api"
)

func main() {
    log.Printf("Server started")
    port := os.Getenv("SAMPLE_AND_DIAGNOSIS_API_PORT")
    if port == "" {
        port = "8080"
    }
    environment := os.Getenv("SAMPLE_AND_DIAGNOSIS_API_ENVIRONMENT")
    if !strings.EqualFold(environment, "production") { // case insensitive comparison
        gin.SetMode(gin.DebugMode)
    }
    engine := gin.New()
    engine.Use(gin.Recovery())
    // request routings
    engine.GET("/openapi", api.HandleOpenApi)
    engine.Run(":" + port)
}
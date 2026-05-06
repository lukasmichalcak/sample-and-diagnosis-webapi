package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lukasmichalcak/sample-and-diagnosis-webapi/api"
	"github.com/lukasmichalcak/sample-and-diagnosis-webapi/internal/db_service"
	"github.com/lukasmichalcak/sample-and-diagnosis-webapi/internal/sample_and_diagnosis_test"
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
	engine.Use(func(ctx *gin.Context) {
		ctx.Header("Access-Control-Allow-Origin", "*")
		ctx.Header("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, PATCH, OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "Origin, Authorization, Content-Type")
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	})

	dbService := db_service.NewMongoService[sample_and_diagnosis_test.Sample](db_service.MongoServiceConfig{})
	defer dbService.Disconnect(context.Background())
	engine.Use(func(ctx *gin.Context) {
		ctx.Set("db_service", dbService)
		ctx.Next()
	})

	// request routings
	handleFunctions := &sample_and_diagnosis_test.ApiHandleFunctions{
		PatientReportsAPI:     sample_and_diagnosis_test.NewPatientReportsAPI(),
		SampleMeasurementsAPI: sample_and_diagnosis_test.NewSampleMeasurementsAPI(),
		SampleReportsAPI:      sample_and_diagnosis_test.NewSampleReportsAPI(),
		SamplesAPI:            sample_and_diagnosis_test.NewSamplesAPI(),
		TestTypesAPI:          sample_and_diagnosis_test.NewTestTypesAPI(),
	}
	sample_and_diagnosis_test.NewRouterWithGinEngine(engine, *handleFunctions)
	engine.GET("/openapi", api.HandleOpenApi)
	engine.Run(":" + port)
}

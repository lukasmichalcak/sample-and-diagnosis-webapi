package sample_and_diagnosis_test

import (
	"net/http"

	"github.com/gin-gonic/gin"
)


type implSampleReportsAPI struct {
}

func NewSampleReportsAPI() SampleReportsAPI{
	return &implSampleReportsAPI{}
}

func (o implSampleReportsAPI) DeleteSampleReport(c *gin.Context) {
    c.AbortWithStatus(http.StatusNotImplemented)
}

func (o implSampleReportsAPI) FinalizeSampleReport(c *gin.Context) {
    c.AbortWithStatus(http.StatusNotImplemented)
}

func (o implSampleReportsAPI) SaveSampleReport(c *gin.Context) {
    c.AbortWithStatus(http.StatusNotImplemented)
}
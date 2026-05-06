package sample_and_diagnosis_test

import (
	"net/http"

	"github.com/gin-gonic/gin"
)


type implSampleMeasurementsAPI struct {
}

func NewSampleMeasurementsAPI() SampleMeasurementsAPI{
	return &implSampleMeasurementsAPI{}
}

func (o implSampleMeasurementsAPI) SaveSampleMeasurements(c *gin.Context) {
    c.AbortWithStatus(http.StatusNotImplemented)
}
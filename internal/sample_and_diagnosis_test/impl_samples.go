package sample_and_diagnosis_test

import (
	"net/http"

	"github.com/gin-gonic/gin"
)


type implSamplesAPI struct {
}

func NewSamplesAPI() SamplesAPI{
	return &implSamplesAPI{}
}

func (o implSamplesAPI) CreateSample(c *gin.Context) {
    c.AbortWithStatus(http.StatusNotImplemented)
}

func (o implSamplesAPI) DeleteSample(c *gin.Context) {
    c.AbortWithStatus(http.StatusNotImplemented)
}

func (o implSamplesAPI) GetSample(c *gin.Context) {
    c.AbortWithStatus(http.StatusNotImplemented)
}

func (o implSamplesAPI) GetSamples(c *gin.Context) {
    c.AbortWithStatus(http.StatusNotImplemented)
}

func (o implSamplesAPI) UpdateSample(c *gin.Context) {
    c.AbortWithStatus(http.StatusNotImplemented)
}

func (o implSamplesAPI) UpdateSampleStatus(c *gin.Context) {
    c.AbortWithStatus(http.StatusNotImplemented)
}
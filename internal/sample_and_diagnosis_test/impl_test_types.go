package sample_and_diagnosis_test

import (
	"net/http"

	"github.com/gin-gonic/gin"
)


type implTestTypesAPI struct {
}

func NewTestTypesAPI() TestTypesAPI{
	return &implTestTypesAPI{}
}

func (o implTestTypesAPI) GetTestTypes(c *gin.Context) {
    c.AbortWithStatus(http.StatusNotImplemented)
}
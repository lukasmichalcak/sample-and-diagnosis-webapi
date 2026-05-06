package sample_and_diagnosis_test

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type implPatientReportsAPI struct {
}

func NewPatientReportsAPI() PatientReportsAPI {
    return &implPatientReportsAPI{}
}

func (o implPatientReportsAPI) GetPatientReports(c *gin.Context) {
    c.AbortWithStatus(http.StatusNotImplemented)
}
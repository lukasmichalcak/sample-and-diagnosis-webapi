package sample_and_diagnosis_test

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type implSampleMeasurementsAPI struct {
}

func NewSampleMeasurementsAPI() SampleMeasurementsAPI {
	return &implSampleMeasurementsAPI{}
}

func (o implSampleMeasurementsAPI) SaveSampleMeasurements(c *gin.Context) {
	updateSampleDocument(c, func(c *gin.Context, sample *Sample) (*Sample, interface{}, int) {
		if sample.Status == DRAFT {
			return nil, gin.H{"status": "Conflict", "message": "Technician draft must be saved before diagnostics", "error": "technician draft must be saved before diagnostics"}, http.StatusConflict
		}
		if sample.Status == FINALIZED {
			return nil, gin.H{"status": "Conflict", "message": "Measurements cannot be changed for finalized documentation", "error": "measurements cannot be changed for finalized documentation"}, http.StatusConflict
		}
		if sample.Status == TAINTED {
			return nil, gin.H{"status": "Conflict", "message": "Measurements cannot be changed for tainted samples", "error": "measurements cannot be changed for tainted samples"}, http.StatusConflict
		}

		var input MeasurementValuesUpdate
		if err := c.ShouldBindJSON(&input); err != nil {
			return nil, gin.H{"status": "Bad Request", "message": "Invalid request body", "error": err.Error()}, http.StatusBadRequest
		}

		measurements, message := normalizedMeasurements(*sample, input.Measurements)
		if message != "" {
			return nil, gin.H{"status": "Bad Request", "message": message, "error": message}, http.StatusBadRequest
		}

		sample.Measurements = measurements
		if sample.Status != REPORT_DRAFT {
			sample.Status = IN_DIAGNOSTICS
		}

		return sample, sample, http.StatusOK
	})
}

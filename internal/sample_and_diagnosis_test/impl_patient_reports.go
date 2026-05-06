package sample_and_diagnosis_test

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type implPatientReportsAPI struct {
}

func NewPatientReportsAPI() PatientReportsAPI {
	return &implPatientReportsAPI{}
}

func (o implPatientReportsAPI) GetPatientReports(c *gin.Context) {
	samples, ok := allSamples(c)
	if !ok {
		return
	}

	patientId := strings.TrimSpace(c.Param("patientId"))
	testTypeFilter := strings.TrimSpace(c.Query("testType"))
	limit := 0
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed < 1 {
			errorResponse(c, http.StatusBadRequest, "Bad Request", "Limit must be a positive integer", "limit must be a positive integer")
			return
		}
		limit = parsed
	}

	result := []Sample{}
	for _, sample := range samples {
		if sample.Status != FINALIZED {
			continue
		}
		if sample.PatientId != patientId {
			continue
		}
		if testTypeFilter != "" && !containsString(sample.TestTypes, testTypeFilter) {
			continue
		}
		result = append(result, sample)
	}

	sortSamplesNewestFirst(result)
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	if len(result) == 0 {
		errorResponse(c, http.StatusNotFound, "Not Found", "Patient documentation does not exist", "patient documentation does not exist")
		return
	}

	c.JSON(http.StatusOK, result)
}

package sample_and_diagnosis_test

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lukasmichalcak/sample-and-diagnosis-webapi/internal/db_service"
)

type implSamplesAPI struct {
}

func NewSamplesAPI() SamplesAPI {
	return &implSamplesAPI{}
}

func (o implSamplesAPI) CreateSample(c *gin.Context) {
	db, ok := sampleDb(c)
	if !ok {
		return
	}

	var input NewSample
	if err := c.ShouldBindJSON(&input); err != nil {
		errorResponse(c, http.StatusBadRequest, "Bad Request", "Invalid request body", err.Error())
		return
	}
	if message := validateNewSample(input); message != "" {
		errorResponse(c, http.StatusBadRequest, "Bad Request", message, message)
		return
	}

	samples, ok := allSamples(c)
	if !ok {
		return
	}
	if sampleCodeExists(samples, input.SampleCode, "") {
		errorResponse(c, http.StatusConflict, "Conflict", "Sample code already exists", "sample code already exists")
		return
	}
	if conflictingPatientName, exists := patientIdentifierNameConflict(samples, input.PatientId, input.PatientName, ""); exists {
		message := "Patient identifier already belongs to " + conflictingPatientName
		errorResponse(c, http.StatusConflict, "Conflict", message, "patient identifier already belongs to another patient name")
		return
	}

	now := time.Now().UTC()
	sample := Sample{
		Id:           newIdentifier("sample"),
		PatientName:  strings.TrimSpace(input.PatientName),
		PatientId:    strings.TrimSpace(input.PatientId),
		SampleCode:   strings.TrimSpace(input.SampleCode),
		CollectedAt:  input.CollectedAt.UTC(),
		TestTypes:    append([]string{}, input.TestTypes...),
		Status:       DRAFT,
		Measurements: emptyMeasurements(input.TestTypes),
		CreatedAt:    now,
		UpdatedAt:    now,
		CreatedBy:    "Lab technician",
	}

	err := db.CreateDocument(c.Request.Context(), sample.Id, &sample)
	switch err {
	case nil:
		c.JSON(http.StatusCreated, sample)
	case db_service.ErrConflict:
		errorResponse(c, http.StatusConflict, "Conflict", "Sample already exists", err.Error())
	default:
		databaseErrorResponse(c, "create sample", err)
	}
}

func (o implSamplesAPI) DeleteSample(c *gin.Context) {
	db, ok := sampleDb(c)
	if !ok {
		return
	}

	sampleId := c.Param("sampleId")
	sample, ok := findSample(c, sampleId)
	if !ok {
		return
	}
	if sample.Status == FINALIZED {
		errorResponse(c, http.StatusConflict, "Conflict", "Finalized sample cannot be deleted", "finalized sample cannot be deleted")
		return
	}

	err := db.DeleteDocument(c.Request.Context(), sampleId)
	switch err {
	case nil:
		c.AbortWithStatus(http.StatusNoContent)
	case db_service.ErrNotFound:
		errorResponse(c, http.StatusNotFound, "Not Found", "Sample not found", err.Error())
	default:
		databaseErrorResponse(c, "delete sample", err)
	}
}

func (o implSamplesAPI) GetSample(c *gin.Context) {
	sample, ok := findSample(c, c.Param("sampleId"))
	if !ok {
		return
	}

	c.JSON(http.StatusOK, sample)
}

func (o implSamplesAPI) GetSamples(c *gin.Context) {
	samples, ok := allSamples(c)
	if !ok {
		return
	}

	statusFilter := parseStatusFilter(c.QueryArray("status"))
	patientIdFilter := strings.TrimSpace(c.Query("patientId"))
	sampleCodeFilter := strings.TrimSpace(c.Query("sampleCode"))
	testTypeFilter := strings.TrimSpace(c.Query("testType"))
	includeTainted, _ := strconv.ParseBool(c.DefaultQuery("includeTainted", "false"))

	filtered := []Sample{}
	for _, sample := range samples {
		if len(statusFilter) > 0 && !statusFilter[sample.Status] {
			continue
		}
		if !includeTainted && sample.Status == TAINTED {
			continue
		}
		if patientIdFilter != "" && sample.PatientId != patientIdFilter {
			continue
		}
		if sampleCodeFilter != "" && !strings.EqualFold(sample.SampleCode, sampleCodeFilter) {
			continue
		}
		if testTypeFilter != "" && !containsString(sample.TestTypes, testTypeFilter) {
			continue
		}
		filtered = append(filtered, sample)
	}

	sortSamplesNewestFirst(filtered)
	c.JSON(http.StatusOK, filtered)
}

func (o implSamplesAPI) UpdateSample(c *gin.Context) {
	samples, ok := allSamples(c)
	if !ok {
		return
	}

	updateSampleDocument(c, func(c *gin.Context, sample *Sample) (*Sample, interface{}, int) {
		if sample.Status != DRAFT {
			return nil, gin.H{
				"status":  "Conflict",
				"message": "Sample is no longer editable by the technician",
				"error":   "sample is no longer editable by the technician",
			}, http.StatusConflict
		}

		var input NewSample
		if err := c.ShouldBindJSON(&input); err != nil {
			return nil, gin.H{"status": "Bad Request", "message": "Invalid request body", "error": err.Error()}, http.StatusBadRequest
		}
		if message := validateNewSample(input); message != "" {
			return nil, gin.H{"status": "Bad Request", "message": message, "error": message}, http.StatusBadRequest
		}

		if sampleCodeExists(samples, input.SampleCode, sample.Id) {
			return nil, gin.H{"status": "Conflict", "message": "Sample code already exists", "error": "sample code already exists"}, http.StatusConflict
		}
		if conflictingPatientName, exists := patientIdentifierNameConflict(samples, input.PatientId, input.PatientName, sample.Id); exists {
			message := "Patient identifier already belongs to " + conflictingPatientName
			return nil, gin.H{"status": "Conflict", "message": message, "error": "patient identifier already belongs to another patient name"}, http.StatusConflict
		}

		sample.PatientName = strings.TrimSpace(input.PatientName)
		sample.PatientId = strings.TrimSpace(input.PatientId)
		sample.SampleCode = strings.TrimSpace(input.SampleCode)
		sample.CollectedAt = input.CollectedAt.UTC()
		sample.TestTypes = append([]string{}, input.TestTypes...)
		sample.Measurements = mergeDraftMeasurements(*sample, input.TestTypes)

		return sample, sample, http.StatusOK
	})
}

func (o implSamplesAPI) UpdateSampleStatus(c *gin.Context) {
	updateSampleDocument(c, func(c *gin.Context, sample *Sample) (*Sample, interface{}, int) {
		var input SampleStatusUpdate
		if err := c.ShouldBindJSON(&input); err != nil {
			return nil, gin.H{"status": "Bad Request", "message": "Invalid request body", "error": err.Error()}, http.StatusBadRequest
		}

		switch input.Status {
		case COLLECTED:
			if sample.Status != DRAFT {
				return nil, gin.H{"status": "Bad Request", "message": "Only technician drafts can be published as collected", "error": "invalid status transition"}, http.StatusBadRequest
			}
			sample.Status = COLLECTED
		case TAINTED:
			if sample.Status == FINALIZED {
				return nil, gin.H{"status": "Conflict", "message": "Finalized documentation cannot be marked tainted", "error": "finalized documentation cannot be marked tainted"}, http.StatusConflict
			}
			sample.Status = TAINTED
		default:
			return nil, gin.H{"status": "Bad Request", "message": "Unsupported status transition", "error": "unsupported status transition"}, http.StatusBadRequest
		}

		return sample, sample, http.StatusOK
	})
}

func parseStatusFilter(rawValues []string) map[SampleStatus]bool {
	filter := map[SampleStatus]bool{}
	for _, raw := range rawValues {
		for _, part := range strings.Split(raw, ",") {
			status := SampleStatus(strings.TrimSpace(part))
			if status != "" {
				filter[status] = true
			}
		}
	}
	return filter
}

func mergeDraftMeasurements(sample Sample, nextTestTypes []string) []MeasurementValue {
	existingByKey := map[string]MeasurementValue{}
	for _, value := range sample.Measurements {
		existingByKey[measurementKey(value.TestTypeCode, value.Code)] = value
	}

	result := emptyMeasurements(nextTestTypes)
	for index, value := range result {
		if existing, ok := existingByKey[measurementKey(value.TestTypeCode, value.Code)]; ok {
			result[index].Value = existing.Value
			result[index].MeasuredAt = existing.MeasuredAt
			result[index].EnteredByRole = existing.EnteredByRole
		}
	}

	return result
}

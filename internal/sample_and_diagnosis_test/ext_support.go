package sample_and_diagnosis_test

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lukasmichalcak/sample-and-diagnosis-webapi/internal/db_service"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const dbServiceContextKey = "db_service"

var supportedTestTypes = []TestType{
	{
		Code:        "crp",
		Name:        "CRP",
		Description: "C-reactive protein inflammation marker.",
		MeasurementSchema: []MeasurementDefinition{
			{Code: "crp_value", Label: "CRP", ValueType: NUMBER, Unit: "mg/L", Min: 0, Required: true},
		},
	},
	{
		Code: "blood_count",
		Name: "Blood count",
		MeasurementSchema: []MeasurementDefinition{
			{Code: "wbc", Label: "White blood cells", ValueType: NUMBER, Unit: "10^9/L", Min: 0, Required: true},
			{Code: "hemoglobin", Label: "Hemoglobin", ValueType: NUMBER, Unit: "g/L", Min: 0},
		},
	},
	{
		Code: "glucose",
		Name: "Glucose",
		MeasurementSchema: []MeasurementDefinition{
			{Code: "glucose_value", Label: "Glucose", ValueType: NUMBER, Unit: "mmol/L", Min: 0, Required: true},
			{Code: "fasting", Label: "Fasting sample", ValueType: BOOLEAN},
		},
	},
	{
		Code: "covid_antigen",
		Name: "COVID antigen",
		MeasurementSchema: []MeasurementDefinition{
			{Code: "result", Label: "Result", ValueType: SELECT, Options: []string{"negative", "positive", "inconclusive"}, Required: true},
		},
	},
	{
		Code: "urine_chemical",
		Name: "Urine chemical",
		MeasurementSchema: []MeasurementDefinition{
			{Code: "protein", Label: "Protein", ValueType: SELECT, Options: []string{"negative", "trace", "positive"}},
			{Code: "notes", Label: "Notes", ValueType: TEXT},
		},
	},
}

func sampleDb(c *gin.Context) (db_service.DbService[Sample], bool) {
	value, exists := c.Get(dbServiceContextKey)
	if !exists {
		errorResponse(c, http.StatusInternalServerError, "Internal Server Error", "db_service not found", "db_service not found")
		return nil, false
	}

	db, ok := value.(db_service.DbService[Sample])
	if !ok {
		errorResponse(c, http.StatusInternalServerError, "Internal Server Error", "db_service context is not of type db_service.DbService", "cannot cast db_service context to db_service.DbService")
		return nil, false
	}

	return db, true
}

func errorResponse(c *gin.Context, status int, label string, message string, detail string) {
	c.JSON(status, gin.H{
		"status":  label,
		"message": message,
		"error":   detail,
	})
}

func databaseErrorResponse(c *gin.Context, action string, err error) {
	errorResponse(c, http.StatusBadGateway, "Bad Gateway", "Failed to "+action+" in database", err.Error())
}

func newIdentifier(prefix string) string {
	var token [6]byte
	if _, err := rand.Read(token[:]); err == nil {
		return fmt.Sprintf("%s-%s", prefix, strings.ToLower(hex.EncodeToString(token[:])))
	}
	return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
}

func allSamples(c *gin.Context) ([]Sample, bool) {
	db, ok := sampleDb(c)
	if !ok {
		return nil, false
	}

	samples, err := db.FindDocuments(c.Request.Context(), bson.D{})
	if err != nil {
		databaseErrorResponse(c, "load samples", err)
		return nil, false
	}

	return samples, true
}

func findSample(c *gin.Context, sampleId string) (*Sample, bool) {
	db, ok := sampleDb(c)
	if !ok {
		return nil, false
	}

	sample, err := db.FindDocument(c.Request.Context(), sampleId)
	switch err {
	case nil:
		return sample, true
	case db_service.ErrNotFound:
		errorResponse(c, http.StatusNotFound, "Not Found", "Sample not found", err.Error())
	default:
		databaseErrorResponse(c, "load sample", err)
	}

	return nil, false
}

type sampleUpdater = func(c *gin.Context, sample *Sample) (updatedSample *Sample, responseContent interface{}, status int)

func updateSampleDocument(c *gin.Context, updater sampleUpdater) {
	db, ok := sampleDb(c)
	if !ok {
		return
	}

	sampleId := c.Param("sampleId")
	sample, err := db.FindDocument(c.Request.Context(), sampleId)
	switch err {
	case nil:
	case db_service.ErrNotFound:
		errorResponse(c, http.StatusNotFound, "Not Found", "Sample not found", err.Error())
		return
	default:
		databaseErrorResponse(c, "load sample", err)
		return
	}

	updatedSample, responseObject, status := updater(c, sample)
	if updatedSample != nil {
		updatedSample.UpdatedAt = time.Now().UTC()
		err = db.UpdateDocument(c.Request.Context(), sampleId, updatedSample)
	} else {
		err = nil
	}

	switch err {
	case nil:
		if responseObject != nil {
			c.JSON(status, responseObject)
		} else {
			c.AbortWithStatus(status)
		}
	case db_service.ErrNotFound:
		errorResponse(c, http.StatusNotFound, "Not Found", "Sample was deleted while processing the request", err.Error())
	default:
		databaseErrorResponse(c, "update sample", err)
	}
}

func validateNewSample(input NewSample) string {
	if strings.TrimSpace(input.PatientName) == "" {
		return "Patient name is required"
	}
	if strings.TrimSpace(input.PatientId) == "" {
		return "Patient identifier is required"
	}
	if strings.TrimSpace(input.SampleCode) == "" {
		return "Sample code is required"
	}
	if input.CollectedAt.IsZero() {
		return "Sample collection time is required"
	}
	if len(input.TestTypes) == 0 {
		return "At least one test type is required"
	}
	for _, testTypeCode := range input.TestTypes {
		if testTypeByCode(testTypeCode) == nil {
			return "Unknown test type: " + testTypeCode
		}
	}
	return ""
}

func testTypeByCode(code string) *TestType {
	for index := range supportedTestTypes {
		if supportedTestTypes[index].Code == code {
			return &supportedTestTypes[index]
		}
	}
	return nil
}

func measurementDefinition(testTypeCode string, code string) *MeasurementDefinition {
	testType := testTypeByCode(testTypeCode)
	if testType == nil {
		return nil
	}

	for index := range testType.MeasurementSchema {
		if testType.MeasurementSchema[index].Code == code {
			return &testType.MeasurementSchema[index]
		}
	}

	return nil
}

func emptyMeasurements(testTypeCodes []string) []MeasurementValue {
	measurements := []MeasurementValue{}
	for _, testTypeCode := range testTypeCodes {
		testType := testTypeByCode(testTypeCode)
		if testType == nil {
			continue
		}
		for _, definition := range testType.MeasurementSchema {
			value := ""
			if definition.ValueType == BOOLEAN {
				value = "false"
			}
			measurements = append(measurements, MeasurementValue{
				TestTypeCode: testTypeCode,
				Code:         definition.Code,
				Value:        value,
				Unit:         definition.Unit,
			})
		}
	}
	return measurements
}

func validateMeasurementValue(value MeasurementValue, definition *MeasurementDefinition) string {
	trimmed := strings.TrimSpace(value.Value)
	if trimmed == "" {
		if definition.Required {
			return definition.Label + " is required"
		}
		return ""
	}

	switch definition.ValueType {
	case NUMBER:
		number, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return definition.Label + " must be a number"
		}
		if number < float64(definition.Min) {
			return definition.Label + " is below the allowed minimum"
		}
		if definition.Max > 0 && number > float64(definition.Max) {
			return definition.Label + " is above the allowed maximum"
		}
	case BOOLEAN:
		if _, err := strconv.ParseBool(trimmed); err != nil {
			return definition.Label + " must be true or false"
		}
	case SELECT:
		for _, option := range definition.Options {
			if trimmed == option {
				return ""
			}
		}
		return definition.Label + " must be one of: " + strings.Join(definition.Options, ", ")
	}

	return ""
}

func normalizedMeasurements(sample Sample, submitted []MeasurementValue) ([]MeasurementValue, string) {
	submittedByKey := map[string]MeasurementValue{}
	for _, value := range submitted {
		definition := measurementDefinition(value.TestTypeCode, value.Code)
		if definition == nil {
			return nil, "Unknown measurement: " + value.TestTypeCode + "/" + value.Code
		}
		if !containsString(sample.TestTypes, value.TestTypeCode) {
			return nil, "Measurement does not belong to this sample test type: " + value.TestTypeCode
		}
		if message := validateMeasurementValue(value, definition); message != "" {
			return nil, message
		}
		value.Unit = definition.Unit
		if strings.TrimSpace(value.EnteredByRole) == "" {
			value.EnteredByRole = "diagnostician"
		}
		if value.MeasuredAt == nil && strings.TrimSpace(value.Value) != "" {
			measuredAt := time.Now().UTC()
			value.MeasuredAt = &measuredAt
		}
		submittedByKey[measurementKey(value.TestTypeCode, value.Code)] = value
	}

	existingByKey := map[string]MeasurementValue{}
	for _, value := range sample.Measurements {
		existingByKey[measurementKey(value.TestTypeCode, value.Code)] = value
	}

	result := []MeasurementValue{}
	for _, testTypeCode := range sample.TestTypes {
		testType := testTypeByCode(testTypeCode)
		if testType == nil {
			continue
		}
		for _, definition := range testType.MeasurementSchema {
			key := measurementKey(testTypeCode, definition.Code)
			value, exists := submittedByKey[key]
			if !exists {
				value, exists = existingByKey[key]
			}
			if !exists {
				value = MeasurementValue{TestTypeCode: testTypeCode, Code: definition.Code, Unit: definition.Unit}
				if definition.ValueType == BOOLEAN {
					value.Value = "false"
				}
			}
			if message := validateMeasurementValue(value, &definition); message != "" {
				return nil, message
			}
			value.Unit = definition.Unit
			result = append(result, value)
		}
	}

	return result, ""
}

func requiredMeasurementsComplete(sample Sample) bool {
	for _, testTypeCode := range sample.TestTypes {
		testType := testTypeByCode(testTypeCode)
		if testType == nil {
			return false
		}
		for _, definition := range testType.MeasurementSchema {
			if !definition.Required {
				continue
			}
			value, ok := sampleMeasurement(sample, testTypeCode, definition.Code)
			if !ok || validateMeasurementValue(value, &definition) != "" {
				return false
			}
		}
	}
	return true
}

func sampleMeasurement(sample Sample, testTypeCode string, code string) (MeasurementValue, bool) {
	for _, value := range sample.Measurements {
		if value.TestTypeCode == testTypeCode && value.Code == code {
			return value, true
		}
	}
	return MeasurementValue{}, false
}

func measurementKey(testTypeCode string, code string) string {
	return testTypeCode + "/" + code
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func sampleCodeExists(samples []Sample, sampleCode string, exceptSampleId string) bool {
	for _, sample := range samples {
		if sample.Id != exceptSampleId && strings.EqualFold(sample.SampleCode, sampleCode) {
			return true
		}
	}
	return false
}

func patientIdentifierNameConflict(samples []Sample, patientId string, patientName string, exceptSampleId string) (string, bool) {
	normalizedPatientId := strings.TrimSpace(patientId)
	normalizedPatientName := normalizePatientName(patientName)
	if normalizedPatientId == "" || normalizedPatientName == "" {
		return "", false
	}

	for _, sample := range samples {
		if sample.Id == exceptSampleId {
			continue
		}
		if strings.TrimSpace(sample.PatientId) == normalizedPatientId &&
			normalizePatientName(sample.PatientName) != normalizedPatientName {
			return strings.TrimSpace(sample.PatientName), true
		}
	}

	return "", false
}

func normalizePatientName(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func sortSamplesNewestFirst(samples []Sample) {
	sort.SliceStable(samples, func(i, j int) bool {
		left := samples[i].UpdatedAt
		right := samples[j].UpdatedAt
		if left.IsZero() {
			left = samples[i].CreatedAt
		}
		if right.IsZero() {
			right = samples[j].CreatedAt
		}
		return left.After(right)
	})
}

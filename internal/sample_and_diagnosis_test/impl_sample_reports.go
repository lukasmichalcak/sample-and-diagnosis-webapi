package sample_and_diagnosis_test

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type implSampleReportsAPI struct {
}

func NewSampleReportsAPI() SampleReportsAPI {
	return &implSampleReportsAPI{}
}

func (o implSampleReportsAPI) DeleteSampleReport(c *gin.Context) {
	updateSampleDocument(c, func(c *gin.Context, sample *Sample) (*Sample, interface{}, int) {
		if sample.Status == FINALIZED {
			return nil, gin.H{"status": "Conflict", "message": "Finalized report cannot be discarded", "error": "finalized report cannot be discarded"}, http.StatusConflict
		}
		if sample.Report == nil || sample.Report.Status != "draft" {
			return nil, gin.H{"status": "Not Found", "message": "Preliminary report does not exist", "error": "preliminary report does not exist"}, http.StatusNotFound
		}

		sample.Report = nil
		if sample.Status != FINALIZED {
			sample.Status = IN_DIAGNOSTICS
		}

		return sample, sample, http.StatusOK
	})
}

func (o implSampleReportsAPI) FinalizeSampleReport(c *gin.Context) {
	updateSampleDocument(c, func(c *gin.Context, sample *Sample) (*Sample, interface{}, int) {
		if sample.Status == FINALIZED {
			return nil, gin.H{"status": "Conflict", "message": "Report is already finalized", "error": "report is already finalized"}, http.StatusConflict
		}
		if sample.Status == TAINTED {
			return nil, gin.H{"status": "Conflict", "message": "Tainted samples cannot be finalized", "error": "tainted samples cannot be finalized"}, http.StatusConflict
		}
		if sample.Report == nil || sample.Report.Status != "draft" {
			return nil, gin.H{"status": "Bad Request", "message": "Preliminary report is required before finalization", "error": "preliminary report is required before finalization"}, http.StatusBadRequest
		}
		if !requiredMeasurementsComplete(*sample) {
			return nil, gin.H{"status": "Bad Request", "message": "Required measurements are missing", "error": "required measurements are missing"}, http.StatusBadRequest
		}

		now := time.Now().UTC()
		sample.Report.Status = "finalized"
		sample.Report.UpdatedAt = now
		sample.Report.FinalizedAt = &now
		sample.Status = FINALIZED
		sample.FinalizedBy = "Diagnostician"

		return sample, sample, http.StatusOK
	})
}

func (o implSampleReportsAPI) SaveSampleReport(c *gin.Context) {
	updateSampleDocument(c, func(c *gin.Context, sample *Sample) (*Sample, interface{}, int) {
		if sample.Status == DRAFT {
			return nil, gin.H{"status": "Conflict", "message": "Technician draft must be saved before diagnostics", "error": "technician draft must be saved before diagnostics"}, http.StatusConflict
		}
		if sample.Status == FINALIZED {
			return nil, gin.H{"status": "Conflict", "message": "Report cannot be changed for finalized documentation", "error": "report cannot be changed for finalized documentation"}, http.StatusConflict
		}
		if sample.Status == TAINTED {
			return nil, gin.H{"status": "Conflict", "message": "Report cannot be changed for tainted samples", "error": "report cannot be changed for tainted samples"}, http.StatusConflict
		}
		if !requiredMeasurementsComplete(*sample) {
			return nil, gin.H{"status": "Bad Request", "message": "Required measurements are missing", "error": "required measurements are missing"}, http.StatusBadRequest
		}

		var input ReportDraft
		if err := c.ShouldBindJSON(&input); err != nil {
			return nil, gin.H{"status": "Bad Request", "message": "Invalid request body", "error": err.Error()}, http.StatusBadRequest
		}
		if strings.TrimSpace(input.Summary) == "" || strings.TrimSpace(input.Conclusion) == "" {
			return nil, gin.H{"status": "Bad Request", "message": "Summary and conclusion are required", "error": "summary and conclusion are required"}, http.StatusBadRequest
		}

		now := time.Now().UTC()
		reportId := newIdentifier("report")
		createdAt := now
		if sample.Report != nil && sample.Report.Id != "" {
			reportId = sample.Report.Id
			createdAt = sample.Report.CreatedAt
		}

		sample.Report = &DiagnosticReport{
			Id:              reportId,
			SampleId:        sample.Id,
			PatientId:       sample.PatientId,
			Summary:         strings.TrimSpace(input.Summary),
			Conclusion:      strings.TrimSpace(input.Conclusion),
			Recommendations: strings.TrimSpace(input.Recommendations),
			CreatedAt:       createdAt,
			UpdatedAt:       now,
			Author:          "Diagnostician",
			Status:          "draft",
		}
		sample.Status = REPORT_DRAFT

		return sample, sample, http.StatusOK
	})
}

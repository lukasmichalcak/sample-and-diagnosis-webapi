package sample_and_diagnosis_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lukasmichalcak/sample-and-diagnosis-webapi/internal/db_service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type SampleAndDiagnosisSuite struct {
	suite.Suite
	dbServiceMock *DbServiceMock[Sample]
}

type DbServiceMock[DocType interface{}] struct {
	mock.Mock
}

func (this *DbServiceMock[DocType]) CreateDocument(ctx context.Context, id string, document *DocType) error {
	args := this.Called(ctx, id, document)
	return args.Error(0)
}

func (this *DbServiceMock[DocType]) FindDocument(ctx context.Context, id string) (*DocType, error) {
	args := this.Called(ctx, id)
	if document := args.Get(0); document != nil {
		return document.(*DocType), args.Error(1)
	}
	return nil, args.Error(1)
}

func (this *DbServiceMock[DocType]) FindDocuments(ctx context.Context, filter interface{}) ([]DocType, error) {
	args := this.Called(ctx, filter)
	if documents := args.Get(0); documents != nil {
		return documents.([]DocType), args.Error(1)
	}
	return nil, args.Error(1)
}

func (this *DbServiceMock[DocType]) UpdateDocument(ctx context.Context, id string, document *DocType) error {
	args := this.Called(ctx, id, document)
	return args.Error(0)
}

func (this *DbServiceMock[DocType]) DeleteDocument(ctx context.Context, id string) error {
	args := this.Called(ctx, id)
	return args.Error(0)
}

func (this *DbServiceMock[DocType]) Disconnect(ctx context.Context) error {
	args := this.Called(ctx)
	return args.Error(0)
}

func TestSampleAndDiagnosisSuite(t *testing.T) {
	suite.Run(t, new(SampleAndDiagnosisSuite))
}

func (suite *SampleAndDiagnosisSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.dbServiceMock = &DbServiceMock[Sample]{}

	var _ db_service.DbService[Sample] = suite.dbServiceMock

	suite.dbServiceMock.
		On("FindDocument", mock.Anything, "test-sample").
		Return(testCollectedSample(), nil)
}

func (suite *SampleAndDiagnosisSuite) Test_SaveSampleMeasurements_DbServiceUpdateCalled() {
	// ARRANGE
	suite.dbServiceMock.
		On("UpdateDocument", mock.Anything, "test-sample", mock.MatchedBy(func(sample *Sample) bool {
			return sample.Status == IN_DIAGNOSTICS &&
				len(sample.Measurements) == 2 &&
				sample.Measurements[0].Value == "5.7" &&
				sample.Measurements[1].Value == "true"
		})).
		Return(nil)

	ctx, recorder := suite.testContext(
		http.MethodPut,
		"/samples/test-sample/measurements",
		`{
			"measurements": [
				{"testTypeCode": "glucose", "code": "glucose_value", "value": "5.7"},
				{"testTypeCode": "glucose", "code": "fasting", "value": "true"}
			]
		}`,
		gin.Param{Key: "sampleId", Value: "test-sample"},
	)
	sut := implSampleMeasurementsAPI{}

	// ACT
	sut.SaveSampleMeasurements(ctx)

	// ASSERT
	suite.Equal(http.StatusOK, recorder.Code)
	suite.dbServiceMock.AssertCalled(suite.T(), "UpdateDocument", mock.Anything, "test-sample", mock.Anything)
}

func (suite *SampleAndDiagnosisSuite) Test_SaveSampleReport_DbServiceUpdateCalled() {
	// ARRANGE
	sample := testCollectedSample()
	now := time.Now().UTC()
	sample.Measurements[0].Value = "5.7"
	sample.Measurements[0].MeasuredAt = &now
	sample.Measurements[1].Value = "true"
	sample.Measurements[1].MeasuredAt = &now
	suite.dbServiceMock.ExpectedCalls = nil
	suite.dbServiceMock.
		On("FindDocument", mock.Anything, "test-sample").
		Return(sample, nil)
	suite.dbServiceMock.
		On("UpdateDocument", mock.Anything, "test-sample", mock.MatchedBy(func(sample *Sample) bool {
			return sample.Status == REPORT_DRAFT &&
				sample.Report != nil &&
				sample.Report.Status == "draft" &&
				sample.Report.Summary == "Glucose is mildly elevated."
		})).
		Return(nil)

	ctx, recorder := suite.testContext(
		http.MethodPut,
		"/samples/test-sample/report",
		`{
			"summary": "Glucose is mildly elevated.",
			"conclusion": "Borderline fasting glucose result.",
			"recommendations": "Repeat fasting glucose check."
		}`,
		gin.Param{Key: "sampleId", Value: "test-sample"},
	)
	sut := implSampleReportsAPI{}

	// ACT
	sut.SaveSampleReport(ctx)

	// ASSERT
	suite.Equal(http.StatusOK, recorder.Code)
	suite.dbServiceMock.AssertCalled(suite.T(), "UpdateDocument", mock.Anything, "test-sample", mock.Anything)
}

func (suite *SampleAndDiagnosisSuite) Test_CreateSample_DbServiceCreateCalled() {
	// ARRANGE
	suite.dbServiceMock.ExpectedCalls = nil
	suite.dbServiceMock.
		On("FindDocuments", mock.Anything, mock.Anything).
		Return([]Sample{}, nil)
	suite.dbServiceMock.
		On("CreateDocument", mock.Anything, mock.Anything, mock.MatchedBy(func(sample *Sample) bool {
			return sample.PatientName == "Eva Novakova" &&
				sample.PatientId == "P-1002" &&
				sample.SampleCode == "SMP-TEST-001" &&
				sample.Status == DRAFT
		})).
		Return(nil)

	ctx, recorder := suite.testContext(
		http.MethodPost,
		"/samples",
		`{
			"patientName": "Eva Novakova",
			"patientId": "P-1002",
			"sampleCode": "SMP-TEST-001",
			"collectedAt": "2026-05-07T09:05:00Z",
			"testTypes": ["glucose"]
		}`,
	)
	sut := implSamplesAPI{}

	// ACT
	sut.CreateSample(ctx)

	// ASSERT
	suite.Equal(http.StatusCreated, recorder.Code)
	suite.dbServiceMock.AssertCalled(suite.T(), "CreateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *SampleAndDiagnosisSuite) Test_CreateSample_RequiresPatientIdentifier() {
	// ARRANGE
	ctx, recorder := suite.testContext(
		http.MethodPost,
		"/samples",
		`{
			"patientName": "Eva Novakova",
			"sampleCode": "SMP-TEST-001",
			"collectedAt": "2026-05-07T09:05:00Z",
			"testTypes": ["glucose"]
		}`,
	)
	sut := implSamplesAPI{}

	// ACT
	sut.CreateSample(ctx)

	// ASSERT
	suite.Equal(http.StatusBadRequest, recorder.Code)
	suite.Contains(recorder.Body.String(), "Patient identifier is required")
	suite.dbServiceMock.AssertNotCalled(suite.T(), "CreateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *SampleAndDiagnosisSuite) Test_CreateSample_RejectsPatientIdentifierNameMismatch() {
	// ARRANGE
	suite.dbServiceMock.ExpectedCalls = nil
	suite.dbServiceMock.
		On("FindDocuments", mock.Anything, mock.Anything).
		Return([]Sample{{
			Id:          "existing-sample",
			PatientName: "Eva Novakova",
			PatientId:   "P-1002",
			SampleCode:  "SMP-EXISTING-001",
			Status:      DRAFT,
		}}, nil)

	ctx, recorder := suite.testContext(
		http.MethodPost,
		"/samples",
		`{
			"patientName": "Juraj Prvy",
			"patientId": "P-1002",
			"sampleCode": "SMP-TEST-002",
			"collectedAt": "2026-05-07T09:05:00Z",
			"testTypes": ["glucose"]
		}`,
	)
	sut := implSamplesAPI{}

	// ACT
	sut.CreateSample(ctx)

	// ASSERT
	suite.Equal(http.StatusConflict, recorder.Code)
	suite.Contains(recorder.Body.String(), "Patient identifier already belongs to Eva Novakova")
	suite.dbServiceMock.AssertNotCalled(suite.T(), "CreateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *SampleAndDiagnosisSuite) Test_UpdateSample_RejectsPatientIdentifierNameMismatch() {
	// ARRANGE
	draft := testCollectedSample()
	draft.Status = DRAFT
	suite.dbServiceMock.ExpectedCalls = nil
	suite.dbServiceMock.
		On("FindDocuments", mock.Anything, mock.Anything).
		Return([]Sample{
			{
				Id:          "existing-sample",
				PatientName: "Eva Novakova",
				PatientId:   "P-1002",
				SampleCode:  "SMP-EXISTING-001",
				Status:      DRAFT,
			},
			*draft,
		}, nil)
	suite.dbServiceMock.
		On("FindDocument", mock.Anything, "test-sample").
		Return(draft, nil)

	ctx, recorder := suite.testContext(
		http.MethodPut,
		"/samples/test-sample",
		`{
			"patientName": "Juraj Prvy",
			"patientId": "P-1002",
			"sampleCode": "SMP-TEST-001",
			"collectedAt": "2026-05-07T09:05:00Z",
			"testTypes": ["glucose"]
		}`,
		gin.Param{Key: "sampleId", Value: "test-sample"},
	)
	sut := implSamplesAPI{}

	// ACT
	sut.UpdateSample(ctx)

	// ASSERT
	suite.Equal(http.StatusConflict, recorder.Code)
	suite.Contains(recorder.Body.String(), "Patient identifier already belongs to Eva Novakova")
	suite.dbServiceMock.AssertNotCalled(suite.T(), "UpdateDocument", mock.Anything, mock.Anything, mock.Anything)
}

func (suite *SampleAndDiagnosisSuite) testContext(method string, target string, body string, params ...gin.Param) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set(dbServiceContextKey, suite.dbServiceMock)
	ctx.Params = params
	ctx.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	return ctx, recorder
}

func testCollectedSample() *Sample {
	return &Sample{
		Id:          "test-sample",
		PatientName: "Eva Novakova",
		PatientId:   "P-1002",
		SampleCode:  "SMP-TEST-001",
		CollectedAt: time.Date(2026, 5, 7, 9, 5, 0, 0, time.UTC),
		TestTypes:   []string{"glucose"},
		Status:      COLLECTED,
		Measurements: []MeasurementValue{
			{TestTypeCode: "glucose", Code: "glucose_value", Unit: "mmol/L", Value: ""},
			{TestTypeCode: "glucose", Code: "fasting", Value: "false"},
		},
		CreatedAt: time.Date(2026, 5, 7, 9, 6, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 5, 7, 9, 6, 0, 0, time.UTC),
		CreatedBy: "Lab technician",
	}
}

package engine

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestProcessInputSyncJSON(t *testing.T) {
	setupTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/process", strings.NewReader(`{"text":"Security incident affecting customer records and urgent remediation."}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ProcessInput(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", w.Code, w.Body.String())
	}

	var body struct {
		Source string           `json:"source"`
		Result ExtractionResult `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Source != "ai" {
		t.Fatalf("expected source ai, got %q", body.Source)
	}
	if body.Result.Urgency == "" {
		t.Fatal("expected urgency to be populated")
	}
	if body.Result.Confidence <= 0 {
		t.Fatal("expected confidence to be populated")
	}
	if body.Result.PossibleCondition == "" {
		t.Fatal("expected possible condition in mock response")
	}
	if len(body.Result.Actions) == 0 {
		t.Fatal("expected actions in mock response")
	}
	if len(body.Result.DoNot) == 0 {
		t.Fatal("expected do_not guidance in mock response")
	}
	if body.Result.Reasoning == "" {
		t.Fatal("expected reasoning in mock response")
	}
}

func TestProcessInputReturnsCacheHit(t *testing.T) {
	setupTestEnv(t)

	inputText := "Quarterly budget review with follow up items."
	expected := &ExtractionResult{
		Urgency:           "MEDIUM",
		Confidence:        0.81,
		Summary:           "cached result",
		PossibleCondition: "Budget overrun / planning variance",
		Actions:           []string{"cached action"},
		DoNot:             []string{"cached caution"},
		Reasoning:         "cached reasoning",
	}
	SetCache(GenerateKey(cacheKeyPayload(&AnalysisInput{Text: inputText})), expected)

	req := httptest.NewRequest(http.MethodPost, "/process", strings.NewReader(`{"text":"`+inputText+`"}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ProcessInput(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", w.Code, w.Body.String())
	}

	var body struct {
		Source string           `json:"source"`
		Result ExtractionResult `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Source != "cache" {
		t.Fatalf("expected source cache, got %q", body.Source)
	}
	if body.Result.Summary != expected.Summary {
		t.Fatalf("expected cached summary %q, got %q", expected.Summary, body.Result.Summary)
	}
}

func TestProcessInputRejectsEmptyInput(t *testing.T) {
	setupTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/process", strings.NewReader(`{"text":"   "}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ProcessInput(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d with body %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "invalid input") {
		t.Fatalf("expected invalid input message, got %s", w.Body.String())
	}
}

func TestProcessInputSSEWithMultipartFileFromFixtures(t *testing.T) {
	setupTestEnv(t)

	req := newMultipartFixtureRequest(t, filepath.Join("..", "..", "test", "cloudy.jpeg"), true)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ProcessInput(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, `"step":"cache"`) {
		t.Fatalf("expected cache event in SSE body, got %s", body)
	}
	if !strings.Contains(body, `"step":"complete"`) {
		t.Fatalf("expected complete event in SSE body, got %s", body)
	}
	if !strings.Contains(body, `"source":"ai"`) {
		t.Fatalf("expected ai source in SSE completion body, got %s", body)
	}
}

func TestProcessInputSSEWithAudioFixture(t *testing.T) {
	setupTestEnv(t)

	req := newMultipartFixtureRequest(t, filepath.Join("..", "..", "test", "arunangshubanerjee-suburban-alley-traffic-ambience-359577.mp3"), true)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ProcessInput(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, `"step":"gemini"`) {
		t.Fatalf("expected gemini event in SSE body, got %s", body)
	}
	if !strings.Contains(body, `"step":"complete"`) {
		t.Fatalf("expected complete event in SSE body, got %s", body)
	}
}

func TestHealthCheckReportsServiceModes(t *testing.T) {
	setupTestEnv(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	HealthCheck(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"vertex_mode":"mock"`) {
		t.Fatalf("expected mock vertex mode, got %s", w.Body.String())
	}
}

func setupTestEnv(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	InitCache()
	bqClient = nil
	gcsClient = nil
	t.Setenv("AXIOM_AI_MODE", "mock")
	t.Setenv("AXIOM_BIGQUERY_MODE", "off")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
}

func newMultipartFixtureRequest(t *testing.T, fixturePath string, stream bool) *http.Request {
	t.Helper()

	fileData, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", fixturePath, err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(fixturePath))
	if err != nil {
		t.Fatalf("failed to create multipart file: %v", err)
	}
	if _, err := part.Write(fileData); err != nil {
		t.Fatalf("failed to write multipart file data: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	target := "/process"
	if stream {
		target += "?stream=true"
	}

	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}

	return req
}

package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ProgressEvent represents a single SSE event
type ProgressEvent struct {
	Step      string      `json:"step"`
	Status    string      `json:"status"`
	Message   string      `json:"message"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// ProcessInput handles both standard API calls and SSE streaming
func ProcessInput(c *gin.Context) {
	// Check if this is an SSE request
	if strings.Contains(c.GetHeader("Accept"), "text/event-stream") || c.Query("stream") == "true" {
		ProcessInputSSE(c)
		return
	}

	// Standard synchronous API
	ProcessInputSync(c)
}

// ProcessInputSync handles standard synchronous processing
func ProcessInputSync(c *gin.Context) {
	inputText, err := extractInputText(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	start := time.Now()
	ctx := context.Background()

	// 2. Cache Lookup
	key := GenerateKey([]byte(inputText))
	if val, found := GetCache(key); found {
		log.Printf("Cache hit for key %s", key)
		result := val.(*ExtractionResult)
		c.JSON(http.StatusOK, gin.H{
			"source":   "cache",
			"result":   result,
			"duration": time.Since(start).String(),
		})
		return
	}

	// 3. Intelligence call
	log.Println("Calling Gemini Flash...")
	result, err := CallGemini(ctx, inputText)
	if err != nil {
		log.Printf("Vertex Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI processing failed", "details": err.Error()})
		return
	}

	// 4. Stream to BigQuery (async-friendly, non-blocking)
	go func() {
		record := &ActionRecord{
			ID:          key,
			Timestamp:   time.Now(),
			InputUri:    fmt.Sprintf("inline://text"),
			Urgency:     result.Urgency,
			Summary:     result.Summary,
			ActionItems: result.ActionItems,
			Entities:    result.Entities,
			Metadata: map[string]interface{}{
				"source": "api",
			},
		}
		if err := StreamToBigQuery(ctx, record); err != nil {
			log.Printf("BigQuery streaming error: %v", err)
		}
	}()

	// 5. Cache and return
	SetCache(key, result)
	c.JSON(http.StatusOK, gin.H{
		"source":   "ai",
		"result":   result,
		"duration": time.Since(start).String(),
	})
}

// ProcessInputSSE handles streaming with Server-Sent Events
func ProcessInputSSE(c *gin.Context) {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	inputText, err := extractInputText(c)
	if err != nil {
		sendErrorEvent(c.Writer, err.Error())
		return
	}

	ctx := context.Background()
	writer := c.Writer

	sendEvent := func(step, status, message string, data interface{}) {
		event := ProgressEvent{
			Step:      step,
			Status:    status,
			Message:   message,
			Timestamp: time.Now(),
			Data:      data,
		}
		eventJSON, _ := json.Marshal(event)
		fmt.Fprintf(writer, "data: %s\n\n", string(eventJSON))
		writer.Flush()
	}

	// 2. Cache Lookup
	key := GenerateKey([]byte(inputText))
	sendEvent("cache", "checking", "Looking up cache...", nil)

	if val, found := GetCache(key); found {
		log.Printf("Cache hit for key %s", key)
		result := val.(*ExtractionResult)
		sendEvent("cache", "hit", "Found in cache", result)
		sendEvent("complete", "success", "Processing complete", gin.H{"source": "cache"})
		return
	}

	sendEvent("cache", "miss", "Not in cache, proceeding...", nil)

	// 3. Intelligence call
	sendEvent("gemini", "processing", "Analyzing with Gemini Flash...", nil)
	result, err := CallGemini(ctx, inputText)
	if err != nil {
		log.Printf("Vertex Error: %v", err)
		sendEvent("gemini", "error", fmt.Sprintf("AI processing failed: %v", err), nil)
		return
	}
	sendEvent("gemini", "complete", "AI analysis complete", result)

	// 4. Stream to BigQuery
	sendEvent("bigquery", "processing", "Saving structured result...", nil)
	record := &ActionRecord{
		ID:          key,
		Timestamp:   time.Now(),
		InputUri:    fmt.Sprintf("inline://text"),
		Urgency:     result.Urgency,
		Summary:     result.Summary,
		ActionItems: result.ActionItems,
		Entities:    result.Entities,
		Metadata: map[string]interface{}{
			"source": "sse",
		},
	}
	if err := StreamToBigQuery(ctx, record); err != nil {
		log.Printf("BigQuery streaming error: %v (non-fatal)", err)
		sendEvent("bigquery", "warning", "BigQuery save skipped (non-critical)", nil)
	} else {
		sendEvent("bigquery", "complete", "Saved to BigQuery", nil)
	}

	// 5. Cache result
	SetCache(key, result)

	// 6. Final response
	sendEvent("complete", "success", "Processing complete", gin.H{
		"source": "ai",
		"result": result,
	})
}

// HealthCheck is a simple health endpoint
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"timestamp": time.Now(),
		"services": gin.H{
			"cache": true,
			"gcs": gcsClient != nil,
			"bigquery": bqClient != nil,
			"vertex_mode": currentVertexMode(),
		},
	})
}

func extractInputText(c *gin.Context) (string, error) {
	var input struct {
		Text string `json:"text"`
	}

	contentType := c.ContentType()
	if strings.Contains(contentType, "application/json") {
		if err := c.ShouldBindJSON(&input); err == nil {
			return validateInputText(input.Text)
		}
	}

	if textForm := c.PostForm("text"); textForm != "" {
		return validateInputText(textForm)
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return "", fmt.Errorf("invalid input: provide non-empty 'text' or 'file'")
		}
		return "", fmt.Errorf("failed to read uploaded file: %w", err)
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read uploaded file: %w", err)
	}

	return normalizeUploadedFile(file, fileBytes)
}

func validateInputText(text string) (string, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", fmt.Errorf("invalid input: provide non-empty 'text' or 'file'")
	}
	return trimmed, nil
}

func sendErrorEvent(writer gin.ResponseWriter, message string) {
	fmt.Fprintf(writer, "data: {\"step\":\"input\",\"status\":\"error\",\"message\":%q}\n\n", message)
	writer.Flush()
}

func normalizeUploadedFile(file multipart.File, fileBytes []byte) (string, error) {
	if len(fileBytes) == 0 {
		return "", fmt.Errorf("uploaded file is empty")
	}

	if seeker, ok := file.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return "", fmt.Errorf("failed to reset uploaded file stream: %w", err)
		}
	}

	contentType := http.DetectContentType(fileBytes)
	if isTextLikeContent(contentType, fileBytes) {
		return validateInputText(string(fileBytes))
	}

	return fmt.Sprintf(
		"Binary file uploaded for analysis. MIME type: %s. Size: %d bytes. Extract high-level operational signals, probable risk, and recommended next steps from this file context.",
		contentType,
		len(fileBytes),
	), nil
}

func isTextLikeContent(contentType string, fileBytes []byte) bool {
	if strings.HasPrefix(contentType, "text/") || contentType == "application/json" || contentType == "application/xml" {
		return true
	}
	return false
}

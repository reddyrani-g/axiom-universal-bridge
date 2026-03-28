package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
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
	if c.GetString("accept") == "text/event-stream" || c.Query("stream") == "true" {
		ProcessInputSSE(c)
		return
	}

	// Standard synchronous API
	ProcessInputSync(c)
}

// ProcessInputSync handles standard synchronous processing
func ProcessInputSync(c *gin.Context) {
	// 1. Read input payload
	var input struct {
		Text string `json:"text"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		textForm := c.PostForm("text")
		if textForm != "" {
			input.Text = textForm
		} else {
			file, _, err := c.Request.FormFile("file")
			if err == nil {
				defer file.Close()
				fileBytes, _ := io.ReadAll(file)
				input.Text = string(fileBytes)
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: provide 'text' or 'file'"})
				return
			}
		}
	}

	start := time.Now()
	ctx := context.Background()

	// 2. Cache Lookup
	key := GenerateKey([]byte(input.Text))
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

	// 3. DLP Redaction
	log.Println("Applying DLP redaction...")
	scrubbedText, err := RedactPII(ctx, input.Text)
	if err != nil {
		log.Printf("DLP error: %v. Continuing with fallback...", err)
		scrubbedText = FakeDLP(input.Text)
	}

	// 4. Intelligence call
	log.Println("Calling Gemini Flash...")
	result, err := CallGemini(ctx, scrubbedText)
	if err != nil {
		log.Printf("Vertex Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI processing failed", "details": err.Error()})
		return
	}

	// 5. Stream to BigQuery (async-friendly, non-blocking)
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

	// 6. Cache and return
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

	// 1. Read input payload
	var input struct {
		Text string `json:"text"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		textForm := c.PostForm("text")
		if textForm != "" {
			input.Text = textForm
		} else {
			file, _, err := c.Request.FormFile("file")
			if err == nil {
				defer file.Close()
				fileBytes, _ := io.ReadAll(file)
				input.Text = string(fileBytes)
			} else {
				c.String(http.StatusBadRequest, "data: {\"error\": \"Invalid input\"}\n\n")
				return
			}
		}
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
	key := GenerateKey([]byte(input.Text))
	sendEvent("cache", "checking", "Looking up cache...", nil)

	if val, found := GetCache(key); found {
		log.Printf("Cache hit for key %s", key)
		result := val.(*ExtractionResult)
		sendEvent("cache", "hit", "Found in cache", result)
		sendEvent("complete", "success", "Processing complete", gin.H{"source": "cache"})
		return
	}

	sendEvent("cache", "miss", "Not in cache, proceeding...", nil)

	// 3. DLP Redaction
	sendEvent("dlp", "processing", "Scanning for PII...", nil)
	scrubbedText, err := RedactPII(ctx, input.Text)
	if err != nil {
		log.Printf("DLP error: %v. Using fallback...", err)
		scrubbedText = FakeDLP(input.Text)
	}
	sendEvent("dlp", "complete", "PII redaction complete", nil)

	// 4. Intelligence call
	sendEvent("gemini", "processing", "Analyzing with Gemini Flash...", nil)
	result, err := CallGemini(ctx, scrubbedText)
	if err != nil {
		log.Printf("Vertex Error: %v", err)
		sendEvent("gemini", "error", fmt.Sprintf("AI processing failed: %v", err), nil)
		return
	}
	sendEvent("gemini", "complete", "AI analysis complete", result)

	// 5. Stream to BigQuery
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

	// 6. Cache result
	SetCache(key, result)

	// 7. Final response
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
	})
}

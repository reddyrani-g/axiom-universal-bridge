package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"cloud.google.com/go/vertexai/genai"
)

type ExtractionResult struct {
	Urgency     string   `json:"urgency"`
	Summary     string   `json:"summary"`
	ActionItems []string `json:"action_items"`
	Entities    []string `json:"entities"`
}

// CallGemini sends the request to the Vertex AI gemini-3-flash model
func CallGemini(ctx context.Context, text string) (*ExtractionResult, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	location := os.Getenv("GOOGLE_CLOUD_LOCATION")
	if location == "" {
		location = "us-central1"
	}

    // Skip true Vertex integration if no project is set (graceful fallback for prototype)
    if projectID == "" {
        fmt.Println("Warning: GOOGLE_CLOUD_PROJECT not set. Returning mock AI result.")
        return &ExtractionResult{
            Urgency:     "HIGH",
            Summary:     "Extracted summary from " + text[:min(len(text), 30)] + "...",
            ActionItems: []string{"Review PII changes", "Execute extracted logic"},
            Entities:    []string{"Simulated", "Artifacts"},
        }, nil
    }

	client, err := genai.NewClient(ctx, projectID, location)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-3-flash")
	model.ResponseMIMEType = "application/json"
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text("You are an expert action extraction engine. Analyze the provided text. Return a JSON object exactly matching this schema: {\"urgency\": \"(LOW/MEDIUM/HIGH/CRITICAL)\", \"summary\": \"(1-2 sentence summary)\", \"action_items\": [\"list\", \"of\", \"actions\"], \"entities\": [\"recognized\", \"entities\"]}"),
		},
	}

	resp, err := model.GenerateContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("model generation failed: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from model")
	}

	part := resp.Candidates[0].Content.Parts[0]
	var result ExtractionResult

    if textPart, ok := part.(genai.Text); ok {
        err = json.Unmarshal([]byte(textPart), &result)
        if err != nil {
            return nil, fmt.Errorf("failed to parse JSON: %v", err)
        }
        return &result, nil
    }

	return nil, fmt.Errorf("unexpected model response format")
}

func min(a, b int) int {
    if a < b { return a }
    return b
}

package engine

import (
	"axiom-bridge/config"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"cloud.google.com/go/vertexai/genai"
)

type ExtractionResult struct {
	Urgency     string   `json:"urgency"`
	Summary     string   `json:"summary"`
	ActionItems []string `json:"action_items"`
	Entities    []string `json:"entities"`
}

// CallGemini sends the request to Vertex AI, or returns a deterministic local mock result.
func CallGemini(ctx context.Context, text string) (*ExtractionResult, error) {
	appConfig := config.Load()
	projectID := appConfig.GoogleCloudProject
	location := appConfig.GoogleCloudLocation

	if useMockAI(projectID) {
		log.Println("Vertex AI unavailable or disabled, returning local mock result")
		return buildMockExtractionResult(text), nil
	}

	client, err := genai.NewClient(ctx, projectID, location)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client: %w", err)
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
		return nil, fmt.Errorf("model generation failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from model")
	}

	part := resp.Candidates[0].Content.Parts[0]
	textPart, ok := part.(genai.Text)
	if !ok {
		return nil, fmt.Errorf("unexpected model response format")
	}

	var result ExtractionResult
	if err := json.Unmarshal([]byte(textPart), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	normalizeExtractionResult(&result, text)
	return &result, nil
}

func useMockAI(projectID string) bool {
	mode := config.Load().AIMode
	return mode == config.AIModeMock || mode == config.AIModeLocal || projectID == ""
}

func currentVertexMode() string {
	if useMockAI(config.Load().GoogleCloudProject) {
		return "mock"
	}
	return "vertex"
}

func buildMockExtractionResult(text string) *ExtractionResult {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return &ExtractionResult{
			Urgency:     "LOW",
			Summary:     "No usable text was provided for analysis.",
			ActionItems: []string{"Provide text input or upload a file to analyze."},
			Entities:    []string{},
		}
	}

	sanitized := strings.Join(strings.Fields(trimmed), " ")
	preview := sanitized
	if len(preview) > 160 {
		preview = preview[:160] + "..."
	}

	result := &ExtractionResult{
		Urgency:     inferUrgency(strings.ToLower(sanitized)),
		Summary:     fmt.Sprintf("Local analysis mode generated a provisional summary from the supplied content: %s", preview),
		ActionItems: buildActionItems(sanitized),
		Entities:    extractEntities(sanitized),
	}

	normalizeExtractionResult(result, text)
	return result
}

func normalizeExtractionResult(result *ExtractionResult, sourceText string) {
	if result.Urgency == "" {
		result.Urgency = inferUrgency(strings.ToLower(sourceText))
	}
	result.Urgency = strings.ToUpper(result.Urgency)

	if strings.TrimSpace(result.Summary) == "" {
		result.Summary = "No summary was returned."
	}

	if len(result.ActionItems) == 0 {
		result.ActionItems = buildActionItems(sourceText)
	}

	if len(result.Entities) == 0 {
		result.Entities = extractEntities(sourceText)
	}
}

func inferUrgency(lowerText string) string {
	switch {
	case strings.Contains(lowerText, "critical"), strings.Contains(lowerText, "breach"), strings.Contains(lowerText, "urgent"), strings.Contains(lowerText, "sev-1"):
		return "CRITICAL"
	case strings.Contains(lowerText, "high"), strings.Contains(lowerText, "incident"), strings.Contains(lowerText, "outage"), strings.Contains(lowerText, "risk"):
		return "HIGH"
	case strings.Contains(lowerText, "medium"), strings.Contains(lowerText, "review"), strings.Contains(lowerText, "follow up"):
		return "MEDIUM"
	default:
		return "LOW"
	}
}

func buildActionItems(text string) []string {
	items := []string{
		"Review the generated summary and validate the urgency classification.",
		"Confirm the extracted entities against the original input.",
	}

	lowerText := strings.ToLower(text)
	switch {
	case strings.Contains(lowerText, "budget"):
		items = append(items, "Review the financial variance and confirm the recommended reallocation.")
	case strings.Contains(lowerText, "incident"), strings.Contains(lowerText, "security"), strings.Contains(lowerText, "breach"):
		items = append(items, "Notify the incident owner and verify remediation steps are tracked.")
	case strings.Contains(lowerText, "audit"), strings.Contains(lowerText, "compliance"):
		items = append(items, "Assign owners to each compliance gap and set remediation deadlines.")
	default:
		items = append(items, "Translate the extracted findings into the next operational step.")
	}

	return items
}

func extractEntities(text string) []string {
	re := regexp.MustCompile(`\b[A-Z][a-zA-Z0-9_-]{2,}\b`)
	matches := re.FindAllString(text, -1)
	if len(matches) == 0 {
		return []string{"LocalMode"}
	}

	seen := map[string]struct{}{}
	entities := make([]string, 0, len(matches))
	for _, match := range matches {
		if _, ok := seen[match]; ok {
			continue
		}
		seen[match] = struct{}{}
		entities = append(entities, match)
		if len(entities) == 6 {
			break
		}
	}

	sort.Strings(entities)
	return entities
}

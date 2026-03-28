package engine

import (
	"axiom-bridge/config"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/vertexai/genai"
)

const emergencyResponsePromptTemplate = `You are an AI-powered emergency response assistant designed to save lives.

Your job is to analyze messy, real-world user input (which may include symptoms, accidents, unclear descriptions, or emotional language) and convert it into structured, actionable, and safety-critical guidance.

IMPORTANT RULES:

* Always prioritize human safety.
* Be cautious and conservative in your assessment.
* If there is any possibility of danger, increase urgency.
* Do NOT provide vague answers.
* Output MUST be valid JSON only (no extra text).

Return the response in the following JSON format:

{
"urgency": "LOW | MEDIUM | HIGH | CRITICAL",
"confidence": number (0 to 1),
"summary": "Clear one-line explanation of the situation",
"possible_condition": "Likely issue (if applicable)",
"actions": [
"Step-by-step immediate actions",
"Keep instructions simple and practical"
],
"do_not": [
"Things the user should avoid doing"
],
"reasoning": "Short explanation of why this assessment was made"
}

GUIDELINES:

* "CRITICAL" = life-threatening (heart attack, unconsciousness, severe bleeding)

* "HIGH" = urgent but not immediately fatal

* "MEDIUM" = needs attention

* "LOW" = safe / informational

* Actions must be:

  * Clear
  * Immediate
  * Practical
  * Non-technical

* If relevant:

  * Suggest calling emergency services
  * Suggest nearby help (generic, no need for real API)

* Avoid hallucinations.

* If unsure, say "possible" or "uncertain" but still guide safely.

INPUT:
%s`

type ExtractionResult struct {
	Urgency           string   `json:"urgency"`
	Confidence        float64  `json:"confidence"`
	Summary           string   `json:"summary"`
	PossibleCondition string   `json:"possible_condition"`
	Actions           []string `json:"actions"`
	DoNot             []string `json:"do_not"`
	Reasoning         string   `json:"reasoning"`
}

// CallGemini sends the request to Vertex AI, or returns a deterministic local mock result.
func CallGemini(ctx context.Context, input *AnalysisInput) (*ExtractionResult, error) {
	appConfig := config.Load()
	projectID := appConfig.GoogleCloudProject
	location := appConfig.GoogleCloudLocation

	if useMockAI(projectID) {
		log.Println("Vertex AI unavailable or disabled, returning local mock result")
		return buildMockExtractionResult(input), nil
	}

	client, err := genai.NewClient(ctx, projectID, location)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-3-flash")
	model.ResponseMIMEType = "application/json"
	parts := buildPromptParts(input)

	resp, err := model.GenerateContent(ctx, parts...)
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

	normalizeExtractionResult(&result, fallbackInputText(input))
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

func buildMockExtractionResult(input *AnalysisInput) *ExtractionResult {
	sourceText := fallbackInputText(input)
	trimmed := strings.TrimSpace(sourceText)
	if trimmed == "" {
		return &ExtractionResult{
			Urgency:           "LOW",
			Confidence:        0.55,
			Summary:           "No usable text was provided for analysis.",
			PossibleCondition: "Insufficient information",
			Actions:           []string{"Provide text input or upload a file to analyze."},
			DoNot:             []string{"Do not rely on this result without usable input."},
			Reasoning:         "The request did not include enough readable content to classify safely.",
		}
	}

	sanitized := strings.Join(strings.Fields(trimmed), " ")
	result := buildScenarioResult(sanitized)

	if input != nil && isMediaMIMEType(input.MIMEType) && strings.TrimSpace(input.Text) == "" {
		result = &ExtractionResult{
			Urgency:           "HIGH",
			Confidence:        0.74,
			Summary:           fmt.Sprintf("A %s file was uploaded and should be reviewed for urgent safety cues.", mediaLabel(input.MIMEType)),
			PossibleCondition: fmt.Sprintf("%s safety incident requiring media review", mediaKind(input.MIMEType)),
			Actions: []string{
				fmt.Sprintf("Inspect the uploaded %s for immediate signs of danger or distress.", mediaLabel(input.MIMEType)),
				"Add a short text description if you want a more accurate safety assessment.",
				"Call emergency services immediately if the evidence suggests a life-threatening situation.",
			},
			DoNot: []string{
				"Do not rely on a mock-mode result alone for a serious emergency.",
			},
			Reasoning: "In mock mode the backend can accept the media upload path, but it cannot truly interpret image or audio content without a live multimodal model call.",
		}
	}

	normalizeExtractionResult(result, sourceText)
	return result
}

func buildPromptParts(input *AnalysisInput) []genai.Part {
	userText := fallbackInputText(input)
	parts := []genai.Part{
		genai.Text(fmt.Sprintf(emergencyResponsePromptTemplate, userText)),
	}

	if input != nil && len(input.FileData) > 0 && isMediaMIMEType(input.MIMEType) {
		parts = append(parts, genai.Blob{
			MIMEType: input.MIMEType,
			Data:     input.FileData,
		})
	}

	return parts
}

func isMediaMIMEType(mimeType string) bool {
	lower := strings.ToLower(mimeType)
	return strings.HasPrefix(lower, "image/") || strings.HasPrefix(lower, "audio/")
}

func mediaKind(mimeType string) string {
	lower := strings.ToLower(mimeType)
	switch {
	case strings.HasPrefix(lower, "audio/"):
		return "audio"
	default:
		return "visual"
	}
}

func mediaLabel(mimeType string) string {
	lower := strings.ToLower(mimeType)
	switch {
	case strings.HasPrefix(lower, "audio/"):
		return "audio"
	default:
		return "image"
	}
}

func normalizeExtractionResult(result *ExtractionResult, sourceText string) {
	if result.Urgency == "" {
		result.Urgency = inferUrgency(strings.ToLower(sourceText))
	}
	result.Urgency = strings.ToUpper(result.Urgency)

	if strings.TrimSpace(result.Summary) == "" {
		result.Summary = "No summary was returned."
	}

	if result.Confidence <= 0 {
		result.Confidence = inferConfidence(result.Urgency)
	}

	if strings.TrimSpace(result.PossibleCondition) == "" {
		result.PossibleCondition = inferCondition(sourceText)
	}

	if len(result.Actions) == 0 {
		result.Actions = buildActions(sourceText)
	}

	if len(result.DoNot) == 0 {
		result.DoNot = buildDoNot(sourceText)
	}

	if strings.TrimSpace(result.Reasoning) == "" {
		result.Reasoning = buildReasoning(sourceText, result.Urgency)
	}
}

func inferUrgency(lowerText string) string {
	switch {
	case strings.Contains(lowerText, "chest pain"), strings.Contains(lowerText, "heart attack"), strings.Contains(lowerText, "difficulty breathing"), strings.Contains(lowerText, "unconscious"), strings.Contains(lowerText, "critical"), strings.Contains(lowerText, "urgent"), strings.Contains(lowerText, "sev-1"):
		return "CRITICAL"
	case strings.Contains(lowerText, "high"), strings.Contains(lowerText, "incident"), strings.Contains(lowerText, "outage"), strings.Contains(lowerText, "risk"), strings.Contains(lowerText, "breach"), strings.Contains(lowerText, "unauthorized access"):
		return "HIGH"
	case strings.Contains(lowerText, "medium"), strings.Contains(lowerText, "review"), strings.Contains(lowerText, "follow up"):
		return "MEDIUM"
	default:
		return "LOW"
	}
}

func buildScenarioResult(text string) *ExtractionResult {
	lowerText := strings.ToLower(text)
	switch {
	case strings.Contains(lowerText, "chest pain") || strings.Contains(lowerText, "sweating") || strings.Contains(lowerText, "dizzy"):
		return &ExtractionResult{
			Urgency:           "CRITICAL",
			Confidence:        0.93,
			Summary:           "Symptoms indicate a possible heart attack.",
			PossibleCondition: "Cardiac event (heart attack)",
			Actions: []string{
				"Call emergency services immediately.",
				"Make the person sit or lie down.",
				"Give aspirin if available and not allergic.",
				"Stay calm and monitor breathing.",
			},
			DoNot: []string{
				"Do not leave the person alone.",
				"Do not delay seeking help.",
			},
			Reasoning: "Chest pain, sweating, and dizziness are classic warning signs of a heart attack.",
		}
	case strings.Contains(lowerText, "budget"):
		return &ExtractionResult{
			Urgency:           "MEDIUM",
			Confidence:        0.84,
			Summary:           "The report points to a budget variance that needs review and possible reallocation.",
			PossibleCondition: "Budget overrun / planning variance",
			Actions: []string{
				"Review the variance drivers against the current budget plan.",
				"Validate whether the overspend is temporary or structural.",
				"Propose a reallocation or corrective spending plan.",
			},
			DoNot: []string{
				"Do not approve new discretionary spend until the variance is understood.",
			},
			Reasoning: "The input describes overspend, variance drivers, and a need to rebalance funds.",
		}
	case strings.Contains(lowerText, "incident"), strings.Contains(lowerText, "security"), strings.Contains(lowerText, "breach"), strings.Contains(lowerText, "unauthorized access"):
		return &ExtractionResult{
			Urgency:           "HIGH",
			Confidence:        0.89,
			Summary:           "The input describes a security incident that requires immediate containment and response coordination.",
			PossibleCondition: "Security incident / potential data breach",
			Actions: []string{
				"Contain the affected systems or credentials immediately.",
				"Notify the incident owner and begin incident response procedures.",
				"Assess scope, impacted records, and remediation status.",
			},
			DoNot: []string{
				"Do not delay escalation to the security response team.",
				"Do not assume the blast radius is limited without verification.",
			},
			Reasoning: "Terms like unauthorized access, incident, breach, and exposed records indicate a potentially serious security event.",
		}
	case strings.Contains(lowerText, "audit"), strings.Contains(lowerText, "compliance"):
		return &ExtractionResult{
			Urgency:           "MEDIUM",
			Confidence:        0.82,
			Summary:           "The input highlights compliance gaps that need tracked remediation.",
			PossibleCondition: "Compliance control deficiency",
			Actions: []string{
				"Assign an owner to each open compliance item.",
				"Set remediation deadlines and dependency tracking.",
				"Prepare evidence for follow-up review.",
			},
			DoNot: []string{
				"Do not treat open audit items as informational only.",
			},
			Reasoning: "Open audit findings and overdue controls usually indicate a remediation management problem rather than a resolved state.",
		}
	default:
		return &ExtractionResult{
			Urgency:           inferUrgency(lowerText),
			Confidence:        inferConfidence(inferUrgency(lowerText)),
			Summary:           fmt.Sprintf("The input suggests a situation that should be reviewed promptly: %s", truncateForSummary(text)),
			PossibleCondition: inferCondition(text),
			Actions:           buildActions(text),
			DoNot:             buildDoNot(text),
			Reasoning:         buildReasoning(text, inferUrgency(lowerText)),
		}
	}
}

func inferConfidence(urgency string) float64 {
	switch urgency {
	case "CRITICAL":
		return 0.93
	case "HIGH":
		return 0.89
	case "MEDIUM":
		return 0.81
	default:
		return 0.72
	}
}

func inferCondition(text string) string {
	lowerText := strings.ToLower(text)
	switch {
	case strings.Contains(lowerText, "chest pain"):
		return "Cardiac event (heart attack)"
	case strings.Contains(lowerText, "security"), strings.Contains(lowerText, "breach"), strings.Contains(lowerText, "unauthorized access"):
		return "Security incident / potential data breach"
	case strings.Contains(lowerText, "budget"):
		return "Budget overrun / planning variance"
	case strings.Contains(lowerText, "audit"), strings.Contains(lowerText, "compliance"):
		return "Compliance control deficiency"
	default:
		return "Needs further assessment"
	}
}

func buildActions(text string) []string {
	lowerText := strings.ToLower(text)
	switch {
	case strings.Contains(lowerText, "chest pain"):
		return []string{
			"Call emergency services immediately.",
			"Keep the person seated or lying down.",
			"Monitor breathing and responsiveness closely.",
		}
	case strings.Contains(lowerText, "security"), strings.Contains(lowerText, "breach"), strings.Contains(lowerText, "incident"):
		return []string{
			"Contain the incident and secure affected systems.",
			"Notify the responsible team and open an incident record.",
			"Assess impact, scope, and required remediation.",
		}
	default:
		return []string{
			"Review the input carefully and confirm the main risk.",
			"Translate the finding into the next concrete action.",
		}
	}
}

func buildDoNot(text string) []string {
	lowerText := strings.ToLower(text)
	switch {
	case strings.Contains(lowerText, "chest pain"):
		return []string{
			"Do not delay seeking emergency help.",
			"Do not leave the person unattended.",
		}
	case strings.Contains(lowerText, "security"), strings.Contains(lowerText, "breach"), strings.Contains(lowerText, "incident"):
		return []string{
			"Do not assume the incident is contained without verification.",
			"Do not delay escalation if customer or regulated data may be affected.",
		}
	default:
		return []string{
			"Do not act on the result without validating the context.",
		}
	}
}

func buildReasoning(text, urgency string) string {
	lowerText := strings.ToLower(text)
	switch {
	case strings.Contains(lowerText, "chest pain"):
		return "Symptoms such as chest pain, sweating, and dizziness can indicate an acute cardiac emergency."
	case strings.Contains(lowerText, "security"), strings.Contains(lowerText, "breach"), strings.Contains(lowerText, "incident"):
		return "The language points to unauthorized activity or possible data exposure, which raises operational and security risk."
	case strings.Contains(lowerText, "budget"):
		return "The described overspend and variance drivers suggest financial controls or planning adjustments are needed."
	default:
		return fmt.Sprintf("The urgency was classified as %s based on the terms and risk indicators present in the input.", strings.ToLower(urgency))
	}
}

func truncateForSummary(text string) string {
	cleaned := strings.Join(strings.Fields(text), " ")
	if len(cleaned) > 140 {
		return cleaned[:140] + "..."
	}
	return cleaned
}

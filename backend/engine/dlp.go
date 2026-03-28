package engine

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"

	dlp "cloud.google.com/go/dlp/apiv2"
	"cloud.google.com/go/dlp/apiv2/dlppb"
)

// DLPClient is a singleton for DLP operations
var dlpClient *dlp.Client

// InitDLP initializes the Cloud DLP client
func InitDLP() error {
	ctx := context.Background()
	client, err := dlp.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create DLP client: %v", err)
	}
	dlpClient = client
	return nil
}

// RedactPII uses Cloud DLP API to redact sensitive information
func RedactPII(ctx context.Context, text string) (string, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Println("Warning: GOOGLE_CLOUD_PROJECT not set. Using fallback redaction.")
		return FakeDLP(text), nil
	}

	if dlpClient == nil {
		log.Println("Warning: DLP client not initialized. Using fallback redaction.")
		return FakeDLP(text), nil
	}

	// Create inspection config for PII detection
	inspectionConfig := &dlppb.InspectConfig{
		InfoTypes: []*dlppb.InfoType{
			{Name: "EMAIL_ADDRESS"},
			{Name: "PERSON_NAME"},
			{Name: "PHONE_NUMBER"},
			{Name: "CREDIT_CARD_NUMBER"},
			{Name: "SOCIAL_SECURITY_NUMBER"},
			{Name: "STREET_ADDRESS"},
		},
	}

	// Create redact config (replace with [REDACTED])
	deidentifyConfig := &dlppb.DeidentifyConfig{
		Transformation: &dlppb.DeidentifyConfig_InfoTypeTransformations{
			InfoTypeTransformations: &dlppb.InfoTypeTransformations{
				Transformations: []*dlppb.InfoTypeTransformations_InfoTypeTransformation{
					{
						InfoTypes: []*dlppb.InfoType{},
						Transformation: &dlppb.InfoTypeTransformations_InfoTypeTransformation_PrimitiveTransformation{
							PrimitiveTransformation: &dlppb.PrimitiveTransformation{
								Transformation: &dlppb.PrimitiveTransformation_ReplaceConfig{
									ReplaceConfig: &dlppb.ReplaceValueConfig{
										NewValue: &dlppb.Value{
											Type: &dlppb.Value_StringValue{
												StringValue: "[REDACTED]",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Create the request
	req := &dlppb.DeidentifyContentRequest{
		Parent: fmt.Sprintf("projects/%s/locations/us", projectID),
		InspectConfig: inspectionConfig,
		DeidentifyConfig: deidentifyConfig,
		Item: &dlppb.ContentItem{
			DataItem: &dlppb.ContentItem_Value{
				Value: text,
			},
		},
	}

	// Call the API
	resp, err := dlpClient.DeidentifyContent(ctx, req)
	if err != nil {
		log.Printf("DLP API error: %v. Falling back to regex redaction.", err)
		return FakeDLP(text), nil
	}

	return resp.Item.Value, nil
}

// CloseDLP closes the DLP client
func CloseDLP() error {
	if dlpClient != nil {
		return dlpClient.Close()
	}
	return nil
}

// FakeDLP is a fallback for local testing (regex-based redaction)
func FakeDLP(input string) string {
	// Scrub emails
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	scrubbed := emailRegex.ReplaceAllString(input, "[EMAIL_REDACTED]")

	// Scrub common SSN format
	ssnRegex := regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	scrubbed = ssnRegex.ReplaceAllString(scrubbed, "[SSN_REDACTED]")

	// Scrub phone numbers (basic)
	phoneRegex := regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`)
	scrubbed = phoneRegex.ReplaceAllString(scrubbed, "[PHONE_REDACTED]")

	// Scrub credit card numbers (basic Luhn-like patterns)
	ccRegex := regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`)
	scrubbed = ccRegex.ReplaceAllString(scrubbed, "[CC_REDACTED]")

	return scrubbed
}

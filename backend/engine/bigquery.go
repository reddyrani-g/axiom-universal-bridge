package engine

import (
	"axiom-bridge/config"
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"
)

// BigQueryClient is a singleton for BigQuery operations
var bqClient *bigquery.Client

// ActionRecord represents a row in BigQuery
type ActionRecord struct {
	ID          string                 `bigquery:"id"`
	Timestamp   time.Time              `bigquery:"timestamp"`
	InputUri    string                 `bigquery:"input_uri"`
	Urgency     string                 `bigquery:"urgency"`
	Summary     string                 `bigquery:"summary"`
	ActionItems []string               `bigquery:"action_items"`
	Entities    []string               `bigquery:"entities"`
	Metadata    map[string]interface{} `bigquery:"metadata"`
}

// InitBigQuery initializes the BigQuery client
func InitBigQuery() error {
	appConfig := config.Load()
	if appConfig.BigQueryMode == config.BigQueryModeOff {
		log.Println("BigQuery disabled by AXIOM_BIGQUERY_MODE=off")
		return nil
	}

	ctx := context.Background()
	projectID := appConfig.GoogleCloudProject
	if projectID == "" {
		return fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable not set")
	}

	client, err := bigquery.NewClient(ctx, projectID, option.WithTelemetryDisabled())
	if err != nil {
		return fmt.Errorf("failed to create BigQuery client: %v", err)
	}
	bqClient = client
	return nil
}

// StreamToBigQuery inserts a record into BigQuery
func StreamToBigQuery(ctx context.Context, record *ActionRecord) error {
	if bqClient == nil {
		return fmt.Errorf("BigQuery client not initialized")
	}

	appConfig := config.Load()
	datasetID := appConfig.BigQueryDataset
	tableID := appConfig.BigQueryTable

	table := bqClient.Dataset(datasetID).Table(tableID)

	// Insert the record
	inserter := table.Inserter()
	if err := inserter.Put(ctx, record); err != nil {
		return fmt.Errorf("failed to insert record into BigQuery: %v", err)
	}

	log.Printf("Inserted record %s into BigQuery", record.ID)
	return nil
}

// CloseBigQuery closes the BigQuery client
func CloseBigQuery() error {
	if bqClient != nil {
		return bqClient.Close()
	}
	return nil
}

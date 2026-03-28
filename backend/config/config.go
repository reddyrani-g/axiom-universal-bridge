package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	DefaultPort             = "8080"
	DefaultGoogleLocation   = "us-central1"
	DefaultGCSBucket        = "axiom-bridge-files"
	DefaultBigQueryDataset  = "axiom_bridge"
	DefaultBigQueryTable    = "actions"
	AIModeMock             = "mock"
	AIModeLocal            = "local"
	BigQueryModeOff         = "off"
	DefaultStreamStepDelayMillis = 250
)

type AppConfig struct {
	Port               string
	GoogleCloudProject string
	GoogleCloudLocation string
	GCSBucket          string
	BigQueryDataset    string
	BigQueryTable      string
	AIMode             string
	BigQueryMode       string
	StreamStepDelayMillis int
}

func Load() AppConfig {
	return AppConfig{
		Port:                getEnv("PORT", DefaultPort),
		GoogleCloudProject:  strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT")),
		GoogleCloudLocation: getEnv("GOOGLE_CLOUD_LOCATION", DefaultGoogleLocation),
		GCSBucket:           getEnv("GCS_BUCKET", DefaultGCSBucket),
		BigQueryDataset:     getEnv("BIGQUERY_DATASET", DefaultBigQueryDataset),
		BigQueryTable:       getEnv("BIGQUERY_TABLE", DefaultBigQueryTable),
		AIMode:              strings.ToLower(strings.TrimSpace(os.Getenv("AXIOM_AI_MODE"))),
		BigQueryMode:        strings.ToLower(strings.TrimSpace(os.Getenv("AXIOM_BIGQUERY_MODE"))),
		StreamStepDelayMillis: getEnvAsInt("AXIOM_STREAM_STEP_DELAY_MS", DefaultStreamStepDelayMillis),
	}
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvAsInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}

	return parsed
}

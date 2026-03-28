#!/bin/bash
set -e

PROJECT_ID=$(gcloud config get-value project)
if [ -z "$PROJECT_ID" ]; then
    echo "ERROR: gcloud project not set. Please run 'gcloud config set project YOUR_PROJECT_ID'"
    exit 1
fi

echo "Deploying Axiom Bridge to Google Cloud Run..."
echo "Project: $PROJECT_ID"

echo "Step 1: Starting deployment from root directory..."

# Deploy using root directory context so Dockerfile can access both frontend and backend
gcloud run deploy axiom-bridge \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars=GOOGLE_CLOUD_PROJECT=$PROJECT_ID,GOOGLE_CLOUD_LOCATION=us-central1,BACKEND_URL=http://localhost:8080 \
  --quiet

echo ""
echo "✅ Axiom Universal Bridge deployed successfully!"
echo ""
echo "Retrieving service URL..."
SERVICE_URL=$(gcloud run services describe axiom-bridge --region us-central1 --format 'value(status.url)')
echo ""
echo "🌉 Service URL: $SERVICE_URL"
echo ""
echo "Access your Bridge at: $SERVICE_URL"


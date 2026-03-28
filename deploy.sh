#!/bin/bash
set -e

PROJECT_ID=$(gcloud config get-value project)
if [ -z "$PROJECT_ID" ]; then
    echo "ERROR: gcloud project not set. Please run 'gcloud config set project axiom-universal-bridge'"
    exit 1
fi

echo "🚀 Deploying Axiom Universal Bridge to Google Cloud Run..."
echo "📦 Project: $PROJECT_ID"

# Deploy Go backend to Cloud Run from backend folder
echo ""
echo "⏳ Building and deploying Go backend..."
cd backend
gcloud run deploy axiom-bridge \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars=GOOGLE_CLOUD_PROJECT=$PROJECT_ID,GOOGLE_CLOUD_LOCATION=us-central1 \
  --quiet
cd ..

echo ""
echo "✅ Axiom Universal Bridge deployed successfully!"
echo ""
echo "🔗 Retrieving service URL..."
SERVICE_URL=$(gcloud run services describe axiom-bridge --region us-central1 --format 'value(status.url)')
echo ""
echo "🌉 Service URL:"
echo "   $SERVICE_URL"
echo ""
echo "📝 Access your Bridge at the URL above"
echo "💬 Frontend runs at http://localhost:3000 during development"
echo "🔌 API proxy available at /api/process on both local and Cloud Run"


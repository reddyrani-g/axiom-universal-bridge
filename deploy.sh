#!/bin/bash
set -e

PROJECT_ID=$(gcloud config get-value project)
if [ -z "$PROJECT_ID" ]; then
    echo "ERROR: gcloud project not set. Please run 'gcloud config set project YOUR_PROJECT_ID'"
    exit 1
fi

echo "Deploying Axiom Bridge pipeline to project: $PROJECT_ID"

echo "1. Building Glassmorphic Frontend Phase..."
cd frontend
npm run build
cd ..

echo "2. Setting up Go Intelligence Backend..."
cd backend
go mod tidy
cd ..

echo "3. Merging Assets (Next.js -> Go Public Directory)..."
rm -rf backend/public
mkdir -p backend/public
cp -r frontend/out/* backend/public/

echo "4. Pushing Payload to Google Cloud Run Serverless..."
cd backend
gcloud run deploy axiom-bridge \
  --source . \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars=GOOGLE_CLOUD_PROJECT=$PROJECT_ID,GOOGLE_CLOUD_LOCATION=us-central1 \
  --quiet

echo "Axiom Universal Bridge Deployment Complete!"

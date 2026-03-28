#!/bin/bash
set -euo pipefail

source "$(dirname "$0")/deploy.config.sh"

PROJECT_ID="$(gcloud config get-value project 2>/dev/null)"
if [[ -z "${PROJECT_ID}" ]]; then
  echo "ERROR: gcloud project not set. Please run 'gcloud config set project axiom-universal-bridge'"
  exit 1
fi

echo "Deploying Axiom Universal Bridge to Google Cloud Run"
echo "Project: ${PROJECT_ID}"
echo "Region: ${GCP_REGION}"

echo ""
echo "Running backend tests before deploy..."
(
  cd backend
  go test ./...
)

echo ""
echo "Deploying backend service..."
(
  cd backend
  gcloud run deploy "${BACKEND_SERVICE_NAME}" \
    --source . \
    --region "${GCP_REGION}" \
    --allow-unauthenticated \
    --set-env-vars="GOOGLE_CLOUD_PROJECT=${PROJECT_ID},GOOGLE_CLOUD_LOCATION=${GCP_REGION},AXIOM_AI_MODE=${BACKEND_AI_MODE},AXIOM_BIGQUERY_MODE=${BACKEND_BIGQUERY_MODE}"
)

BACKEND_URL="$(gcloud run services describe "${BACKEND_SERVICE_NAME}" --region "${GCP_REGION}" --format='value(status.url)')"

echo ""
echo "Type-checking frontend before deploy..."
(
  cd frontend
  npx tsc --noEmit
)

echo ""
echo "Deploying frontend service..."
(
  cd frontend
  gcloud run deploy "${FRONTEND_SERVICE_NAME}" \
    --source . \
    --region "${GCP_REGION}" \
    --allow-unauthenticated \
    --set-env-vars="BACKEND_URL=${BACKEND_URL}"
)

FRONTEND_URL="$(gcloud run services describe "${FRONTEND_SERVICE_NAME}" --region "${GCP_REGION}" --format='value(status.url)')"

echo ""
echo "Deployment complete"
echo "Backend URL: ${BACKEND_URL}"
echo "Frontend URL: ${FRONTEND_URL}"

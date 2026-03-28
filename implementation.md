Axiom Universal Bridge Architecture Plan

This document outlines the architecture and implementation plan for the Axiom Universal Bridge, a multimodal ingestion engine on Google Cloud.



User Review Required

IMPORTANT

I have updated the plan to use a Go backend and incorporated your feedback regarding scalability, efficiency, security, and UI. Please review these improvements before we begin execution.

Architectural Improvements (Scalability, Security, UI)

Based on your feedback, the following industry-standard improvements have been integrated:

Performance (Go): Swapped Python/FastAPI for Go (using the standard net/http or lightweight gin router). This ensures maximum throughput, minimal footprint, and highly concurrent processing via goroutines.

Security (WAF & Auth): Added Cloud Armor in front of Cloud Run to prevent DDoS attacks and enforce rate limiting. Added API Key validation on the Go API to restrict unauthorized access.

Efficiency & Scalability:Connection Pooling: Global singletons for GCP clients (BigQuery, GCS, Vertex) to avoid connection churn.

Streaming/Chunking: Large files will be efficiently pushed to GCS to avoid OOM issues in Cloud Run.

Top-Notch UI: The Next.js frontend will use Tailwind CSS and Framer Motion for premium fluid animations and a glowing glassmorphic aesthetic. We will implement Server-Sent Events (SSE) to stream real-time progress to the UI (e.g., "Scanning for PII...", "Analyzing Image...", "Saving to BigQuery...").

Architecture Overview

The system consists of the following components:

Frontend: A Next.js web application with a glassmorphic UI, Framer Motion animations, and SSE for real-time visualization.

API Gateway / Orchestrator: Cloud Run hosting a highly concurrent Go backend.

Storage:Google Cloud Storage (GCS) for landing raw unstructured files.

BigQuery for storing structured JSON actions.

Security & Safety:Cloud Armor restricting malicious traffic.

Google Cloud DLP for redaction of PII from text inputs.

Model Armor to prevent prompt injections.

Intelligence Engines:Vertex AI via gemini-3-flash (text, audio).

Vertex AI via nano-banana-2 (image reasoning).

Architecture Diagram

Google Cloud Platform

Submit File + Prompt

1. Store Raw File

URI Reference

2. Check for PII

Redacted Text

3. Validate Prompt

Clean Prompt

Text/Audio

Images

JSON Response

JSON Response

4. Stream Row

SSE Progress Updates

Return Structured JSON

User InterfaceNext.js / Glassmorphic

Cloud ArmorWAF / DDoS Protection

Go Backend on Cloud Run/process Endpoint

Cloud Storage

Cloud DLP API

Model Armor

Modality Router

Vertex AIgemini-3-flash

Vertex AInano-banana-2

BigQuery

Proposed Changes

Infrastructure (Terraform)

terraform/main.tf: Define GCP provider and project details.

terraform/storage.tf: GCS bucket for raw files.

terraform/bigquery.tf: BigQuery dataset and table (urgency, summary, action_items, entities).

terraform/cloudrun.tf: Cloud Run service configuration for the Go backend.

terraform/security.tf: Cloud Armor policies.

terraform/iam.tf: Service account with roles for DLP, Vertex AI, GCS, and BigQuery.

Backend (Go)

backend/go.mod: Go module definition.

backend/main.go: Application entry point, routing (/health, /process), middleware (auth, logging).

backend/nexus/engine.go: Core logic orchestrating GCS uploads, DLP redaction, Model Armor checks, Vertex AI multimodal calls (using Google Cloud Go SDKs), and BigQuery streaming.

backend/Dockerfile: Minimal distroless or alpine containerization for the compiled Go binary.

backend/deploy.sh: Script to build and deploy to Cloud Run.

Frontend (React/Next.js)

frontend/: A Next.js project using TailwindCSS and Framer Motion. Features a premium drag-and-drop zone, dynamic progress indicators via SSE, and an interactive JSON viewer.

Verification Plan

Automated Tests

Go unit tests for parsing Vertex AI JSON output.

Manual Verification

Deploy infrastructure with Terraform.

Push the Docker image and deploy to Google Cloud Run.

Test the Next.js visualizer with multimodal inputs and ensure smooth animations and SSE functionality.


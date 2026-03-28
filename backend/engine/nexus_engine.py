import logging
import asyncio
import os
from typing import Optional
from fastapi import UploadFile
from pydantic import BaseModel
import io

from google.cloud import aiplatform
from google.cloud.aiplatform.gapic.types import content
from google.cloud import storage
from google.cloud import bigquery
from google.cloud.dlp_v2 import DlpServiceClient

logger = logging.getLogger(__name__)

class ProcessResponse(BaseModel):
    status: str
    urgency: str
    summary: str
    action_items: list
    entities: list
    raw_file_path: Optional[str] = None
    message: Optional[str] = None

class NexusEngine:
    """Multimodal chaos-to-action converter using Vertex AI"""
    
    def __init__(self, project_id: str):
        self.project_id = project_id
        self.location = os.getenv("GCP_REGION", "us-central1")
        
        # Initialize Vertex AI client
        aiplatform.init(project=project_id, location=self.location)
        
        # Initialize GCS client
        self.storage_client = storage.Client(project=project_id)
        self.bucket_name = os.getenv("GCS_BUCKET", f"{project_id}-bridge-files")
        
        # Initialize BigQuery client
        self.bq_client = bigquery.Client(project=project_id)
        self.dataset_id = os.getenv("BQ_DATASET", "universal_bridge")
        self.table_id = "process_results"
        
        # Initialize DLP client
        self.dlp_client = DlpServiceClient()
        
        logger.info(f"✓ NexusEngine ready: {self.project_id} @ {self.location}")
    
    async def process(
        self,
        text_input: Optional[str] = None,
        file: Optional[UploadFile] = None,
        user_context: Optional[str] = None
    ) -> ProcessResponse:
        """
        Process multimodal input through Vertex AI.
        Returns structured JSON with urgency, summary, action items, and entities.
        """
        
        try:
            # 1. Handle file upload (store raw, handle DLP redaction)
            raw_file_path = None
            file_content = None
            file_uri = None
            
            if file:
                file_content = await file.read()
                raw_file_path = await self._store_raw_file(file.filename, file_content)
                file_uri = f"gs://{self.bucket_name}/{raw_file_path}"
                logger.info(f"✓ File stored: {file_uri}")
            
            # 2. Redact PII using Cloud DLP
            if text_input:
                text_input = await self._redact_pii(text_input)
            
            # 3. Build multimodal request for Vertex AI
            system_instruction = self._get_system_instruction()
            
            # Call Gemini 3 Flash via Vertex AI
            response = await self._call_vertex_ai(
                text_input=text_input,
                file_uri=file_uri,
                file_type=file.content_type if file else None,
                user_context=user_context,
                system_instruction=system_instruction
            )
            
            # 4. Parse and structure response
            result = self._parse_response(response)
            
            # 5. Stream to BigQuery
            await self._stream_to_bigquery(result, raw_file_path)
            
            return result
        
        except Exception as e:
            logger.error(f"✗ Processing failed: {e}")
            raise
    
    async def _store_raw_file(self, filename: str, content: bytes) -> str:
        """Store raw file in GCS"""
        try:
            # Ensure bucket exists
            try:
                bucket = self.storage_client.bucket(self.bucket_name)
                bucket.create()
            except:
                bucket = self.storage_client.bucket(self.bucket_name)
            
            # Upload file
            blob = bucket.blob(f"raw/{filename}")
            blob.upload_from_string(content)
            
            return f"raw/{filename}"
        except Exception as e:
            logger.error(f"✗ GCS storage error: {e}")
            raise
    
    async def _redact_pii(self, text: str) -> str:
        """Redact PII using Cloud DLP"""
        try:
            parent = self.dlp_client.common_project_path(self.project_id)
            
            inspect_config = {
                "info_types": [
                    {"name": "EMAIL_ADDRESS"},
                    {"name": "PHONE_NUMBER"},
                    {"name": "CREDIT_CARD_NUMBER"},
                    {"name": "US_SOCIAL_SECURITY_NUMBER"},
                ],
                "min_likelihood": "LIKELY",
            }
            
            deidentify_config = {
                "info_type_transformations": {
                    "transformations": [
                        {
                            "primitive_transformation": {
                                "replace_with_info_type_config": {}
                            }
                        }
                    ]
                }
            }
            
            response = self.dlp_client.deidentify_content(
                request={
                    "parent": parent,
                    "inspect_config": inspect_config,
                    "deidentify_config": deidentify_config,
                    "item": {"value": text},
                }
            )
            
            return response.item.value
        except Exception as e:
            logger.warning(f"⚠ DLP redaction skipped: {e}")
            return text
    
    async def _call_vertex_ai(
        self,
        text_input: Optional[str],
        file_uri: Optional[str],
        file_type: Optional[str],
        user_context: Optional[str],
        system_instruction: str
    ) -> str:
        """Call Gemini 3 Flash via Vertex AI SDK"""
        try:
            model = aiplatform.GenerativeModel(
                model_name="gemini-3-flash",
                system_instruction=system_instruction
            )
            
            # Build message parts
            parts = []
            
            if text_input:
                parts.append({"text": text_input})
            
            if file_uri:
                # Determine MIME type
                mime_map = {
                    "audio/mpeg": "audio/mpeg",
                    "audio/wav": "audio/wav",
                    "image/jpeg": "image/jpeg",
                    "image/png": "image/png",
                    "application/pdf": "application/pdf",
                }
                mime_type = mime_map.get(file_type, "application/octet-stream")
                
                part = content.Part.from_uri(
                    uri=file_uri,
                    mime_type=mime_type
                )
                parts.append(part)
            
            if user_context:
                parts.append({"text": f"Context: {user_context}"})
            
            # Call model
            response = await asyncio.to_thread(
                model.generate_content,
                parts,
                generation_config=aiplatform.GenerationConfig(
                    temperature=0.7,
                    top_p=0.9,
                    max_output_tokens=2048,
                )
            )
            
            return response.text
        except Exception as e:
            logger.error(f"✗ Vertex AI error: {e}")
            raise
    
    def _get_system_instruction(self) -> str:
        """High-fidelity system instruction enforcing JSON schema"""
        return """You are the Axiom Universal Bridge - a sophisticated multimodal intelligence engine.

Your task: Convert unstructured chaos (text, audio transcripts, image descriptions) into actionable intelligence.

RESPOND ONLY WITH VALID JSON in this exact schema:
{
    "urgency": "CRITICAL|HIGH|MEDIUM|LOW",
    "summary": "2-3 sentence executive summary",
    "action_items": ["specific actionable step 1", "specific actionable step 2", ...],
    "entities": ["person", "organization", "location", "concept", ...]
}

CRITICAL RULES:
1. Always output valid JSON - no markdown, no code blocks, just raw JSON
2. Urgency: CRITICAL only if time-sensitive/life-threatening, HIGH for immediate action, MEDIUM for important, LOW for reference
3. Action items must be specific, measurable, and immediately executable
4. Extract all named entities (people, organizations, locations, important concepts)
5. Be concise but complete - max 3 action items unless more are genuinely needed

Example output:
{"urgency":"HIGH","summary":"Customer reports production outage affecting 10K users. Database replication failed.","action_items":["Escalate to DB team immediately","Check replication lag on primary","Activate failover protocol"],"entities":["Production DB","Customer","10K users"]}"""
    
    def _parse_response(self, response_text: str) -> ProcessResponse:
        """Parse Vertex AI JSON response"""
        import json
        
        try:
            # Extract JSON from response
            data = json.loads(response_text)
            
            return ProcessResponse(
                status="success",
                urgency=data.get("urgency", "MEDIUM"),
                summary=data.get("summary", ""),
                action_items=data.get("action_items", []),
                entities=data.get("entities", [])
            )
        except json.JSONDecodeError as e:
            logger.error(f"✗ JSON parse error: {e}")
            return ProcessResponse(
                status="error",
                urgency="MEDIUM",
                summary="Failed to parse response",
                action_items=[],
                entities=[],
                message=str(e)
            )
    
    async def _stream_to_bigquery(self, result: ProcessResponse, file_path: Optional[str]):
        """Stream structured result to BigQuery"""
        try:
            table_id = f"{self.project_id}.{self.dataset_id}.{self.table_id}"
            table = self.bq_client.get_table(table_id)
            
            row = {
                "timestamp": asyncio.get_event_loop().time(),
                "urgency": result.urgency,
                "summary": result.summary,
                "action_items": result.action_items,
                "entities": result.entities,
                "raw_file_path": file_path,
            }
            
            errors = self.bq_client.insert_rows_json(table, [row])
            if errors:
                logger.warning(f"⚠ BigQuery insert warnings: {errors}")
            else:
                logger.info(f"✓ Streamed to BigQuery: {table_id}")
        except Exception as e:
            logger.warning(f"⚠ BigQuery streaming skipped: {e}")

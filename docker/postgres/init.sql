-- Action-Based Execution Tracing Schema
-- Hybrid approach: Metadata in PostgreSQL, Payloads in S3
--
-- ⚠️  DATABASE SEPARATION - READ THIS! ⚠️
--
-- This schema MUST be applied to the action_traces database ONLY!
-- Do NOT mix this with other databases:
--
--   ✓ action_traces  - Distributed action execution tracing (THIS SCHEMA)
--   ✗ when_metrics   - WHEN workflow execution metrics (different schema)
--   ✗ claude_metrics - EVE service metrics (different schema)
--
-- Usage:
--   psql -U claude -d action_traces -f action_tracing_schema.sql
--
-- Purpose:
--   Track distributed actions across ALL EVE services with:
--   - Correlation IDs linking actions in workflows
--   - S3 storage for request/response payloads
--   - Queryable metadata for debugging
--   - TimescaleDB for time-series queries

-- Enable TimescaleDB and UUID extensions
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- Action Executions Table (Lightweight Metadata)
-- ============================================================================

CREATE TABLE IF NOT EXISTS action_executions (
    id UUID DEFAULT uuid_generate_v4(),

    -- Correlation tracking (links related actions in a workflow)
    correlation_id TEXT NOT NULL,
    operation_id TEXT NOT NULL,
    parent_operation_id TEXT,  -- For nested/chained actions

    -- Action identity (from JSON-LD @type fields)
    action_type TEXT NOT NULL,         -- @type: "CreateAction", "TransferAction", etc.
    object_type TEXT,                  -- object.@type: "SoftwareApplication", "Database", etc.
    target_type TEXT,                  -- target.@type (if present)
    instrument_type TEXT,              -- instrument.@type (if present)
    action_context TEXT DEFAULT 'https://schema.org',  -- @context

    -- Service information
    service_id TEXT NOT NULL,
    endpoint TEXT,
    http_method TEXT,

    -- Timing
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms BIGINT,

    -- Status
    action_status TEXT,  -- "CompletedActionStatus", "FailedActionStatus", "ActiveActionStatus"
    error_message TEXT,
    error_type TEXT,

    -- S3 Storage References (URLs, NOT the actual data)
    request_url TEXT,      -- s3://eve-traces/{correlation_id}/{operation_id}/request.json
    response_url TEXT,     -- s3://eve-traces/{correlation_id}/{operation_id}/response.json
    logs_url TEXT,         -- s3://eve-traces/{correlation_id}/{operation_id}/logs.txt
    artifacts_url TEXT,    -- s3://eve-traces/{correlation_id}/{operation_id}/artifacts/

    -- Size tracking (for monitoring)
    request_size_bytes BIGINT DEFAULT 0,
    response_size_bytes BIGINT DEFAULT 0,
    logs_size_bytes BIGINT DEFAULT 0,

    -- Queryable metadata (small, extracted fields for filtering)
    -- This is action-type specific and varies based on action+object combination
    metadata JSONB DEFAULT '{}'::jsonb,

    -- Request context
    client_ip INET,
    user_agent TEXT,

    -- Tags for custom categorization
    tags TEXT[] DEFAULT '{}',

    -- OpenTelemetry integration (links to technical traces)
    otel_trace_id TEXT,  -- OpenTelemetry trace ID (32-char hex)
    otel_span_id TEXT,   -- OpenTelemetry span ID (16-char hex)

    PRIMARY KEY (started_at, correlation_id, operation_id)
);

-- Create hypertable for time-series optimization
SELECT create_hypertable(
    'action_executions',
    'started_at',
    if_not_exists => TRUE,
    chunk_time_interval => INTERVAL '1 day'
);

-- ============================================================================
-- Indexes for Fast Queries
-- ============================================================================

-- Query by correlation (get all actions in a workflow)
CREATE INDEX IF NOT EXISTS idx_action_exec_correlation
    ON action_executions (correlation_id, started_at DESC);

-- Query by operation ID (get specific action)
CREATE INDEX IF NOT EXISTS idx_action_exec_operation
    ON action_executions (operation_id);

-- Query by service (get all actions for a service)
CREATE INDEX IF NOT EXISTS idx_action_exec_service
    ON action_executions (service_id, started_at DESC);

-- Query by action type (get all CreateActions, etc.)
CREATE INDEX IF NOT EXISTS idx_action_exec_action_type
    ON action_executions (action_type, started_at DESC);

-- Query by action + object type combination
CREATE INDEX IF NOT EXISTS idx_action_exec_type_combo
    ON action_executions (action_type, object_type, started_at DESC);

-- Query by status (get failed actions)
CREATE INDEX IF NOT EXISTS idx_action_exec_status
    ON action_executions (action_status, started_at DESC);

-- Query by parent (get child actions)
CREATE INDEX IF NOT EXISTS idx_action_exec_parent
    ON action_executions (parent_operation_id, started_at DESC)
    WHERE parent_operation_id IS NOT NULL;

-- GIN index for metadata JSONB queries
CREATE INDEX IF NOT EXISTS idx_action_exec_metadata
    ON action_executions USING GIN (metadata);

-- GIN index for tags array
CREATE INDEX IF NOT EXISTS idx_action_exec_tags
    ON action_executions USING GIN (tags);

-- Index for OpenTelemetry trace ID (link semantic → OTel)
CREATE INDEX IF NOT EXISTS idx_action_exec_otel_trace
    ON action_executions (otel_trace_id)
    WHERE otel_trace_id IS NOT NULL;

-- ============================================================================
-- Action Metadata Schema Registry
-- ============================================================================
-- Defines what metadata fields should be extracted per action type

CREATE TABLE IF NOT EXISTS action_metadata_schemas (
    id SERIAL PRIMARY KEY,
    action_type TEXT NOT NULL,
    object_type TEXT NOT NULL,
    schema_definition JSONB NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(action_type, object_type)
);

-- Insert common action metadata schemas
INSERT INTO action_metadata_schemas (action_type, object_type, schema_definition, description) VALUES

-- Container creation
('CreateAction', 'SoftwareApplication',
'{
    "fields": {
        "container_id": {"type": "string", "description": "Docker/Podman container ID"},
        "image": {"type": "string", "description": "Container image name"},
        "started": {"type": "boolean", "description": "Whether container was started"},
        "ports": {"type": "array", "description": "Port mappings"},
        "health_status": {"type": "string", "description": "Container health status"}
    },
    "required": ["container_id", "image"]
}'::jsonb,
'Container creation metadata'),

-- Database migration
('TransferAction', 'Database',
'{
    "fields": {
        "source_database": {"type": "string", "description": "Source database name"},
        "target_database": {"type": "string", "description": "Target database name"},
        "migration_type": {"type": "string", "description": "full, incremental, schema_only"},
        "total_tables": {"type": "number", "description": "Total tables to migrate"},
        "completed_tables": {"type": "number", "description": "Tables migrated so far"},
        "total_rows": {"type": "number", "description": "Total rows to migrate"},
        "transferred_rows": {"type": "number", "description": "Rows migrated so far"},
        "progress_percent": {"type": "number", "description": "Migration progress 0-100"},
        "current_table": {"type": "string", "description": "Currently migrating table"},
        "verification_pending": {"type": "boolean", "description": "Whether verification is pending"}
    },
    "required": ["source_database", "target_database", "progress_percent"]
}'::jsonb,
'Database migration metadata'),

-- Backup operations
('UploadAction', 'Dataset',
'{
    "fields": {
        "backup_type": {"type": "string", "description": "full, incremental, differential"},
        "source_database": {"type": "string", "description": "Database being backed up"},
        "backup_size_bytes": {"type": "number", "description": "Backup size in bytes"},
        "compression_type": {"type": "string", "description": "Compression algorithm used"},
        "checksum": {"type": "string", "description": "Backup file checksum"},
        "storage_location": {"type": "string", "description": "S3 path or file location"},
        "retention_days": {"type": "number", "description": "How long to keep backup"},
        "expires_at": {"type": "datetime", "description": "When backup expires"}
    },
    "required": ["backup_type", "storage_location", "checksum"]
}'::jsonb,
'Backup operation metadata'),

-- ETL transformations
('ReplaceAction', 'DataFeed',
'{
    "fields": {
        "source_tables": {"type": "array", "description": "Source tables"},
        "destination_table": {"type": "string", "description": "Destination table"},
        "input_rows": {"type": "number", "description": "Input row count"},
        "output_rows": {"type": "number", "description": "Output row count"},
        "filtered_rows": {"type": "number", "description": "Rows filtered out"},
        "rows_per_second": {"type": "number", "description": "Processing throughput"},
        "data_quality_passed": {"type": "boolean", "description": "Whether quality checks passed"}
    },
    "required": ["input_rows", "output_rows"]
}'::jsonb,
'ETL transformation metadata'),

-- CI/CD builds
('ExecuteAction', 'SoftwareSourceCode',
'{
    "fields": {
        "repository": {"type": "string", "description": "Git repository"},
        "branch": {"type": "string", "description": "Git branch"},
        "commit_sha": {"type": "string", "description": "Git commit SHA"},
        "build_number": {"type": "number", "description": "Build number"},
        "pipeline_name": {"type": "string", "description": "Pipeline name"},
        "tests_passed": {"type": "number", "description": "Number of tests passed"},
        "tests_failed": {"type": "number", "description": "Number of tests failed"},
        "artifacts_count": {"type": "number", "description": "Number of artifacts generated"}
    },
    "required": ["repository", "commit_sha"]
}'::jsonb,
'CI/CD build metadata')

ON CONFLICT (action_type, object_type) DO NOTHING;

-- ============================================================================
-- Helper Functions
-- ============================================================================

-- Get full workflow trace (all actions for a correlation ID)
CREATE OR REPLACE FUNCTION get_workflow_trace(p_correlation_id TEXT)
RETURNS TABLE (
    operation_id TEXT,
    parent_operation_id TEXT,
    action_type TEXT,
    object_type TEXT,
    service_id TEXT,
    started_at TIMESTAMPTZ,
    duration_ms BIGINT,
    action_status TEXT,
    request_url TEXT,
    response_url TEXT,
    metadata JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        ae.operation_id,
        ae.parent_operation_id,
        ae.action_type,
        ae.object_type,
        ae.service_id,
        ae.started_at,
        ae.duration_ms,
        ae.action_status,
        ae.request_url,
        ae.response_url,
        ae.metadata
    FROM action_executions ae
    WHERE ae.correlation_id = p_correlation_id
    ORDER BY ae.started_at ASC;
END;
$$ LANGUAGE plpgsql;

-- Get failed actions for a service
CREATE OR REPLACE FUNCTION get_failed_actions(
    p_service_id TEXT,
    p_hours INTEGER DEFAULT 24
)
RETURNS TABLE (
    operation_id TEXT,
    correlation_id TEXT,
    action_type TEXT,
    started_at TIMESTAMPTZ,
    duration_ms BIGINT,
    error_message TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        ae.operation_id,
        ae.correlation_id,
        ae.action_type,
        ae.started_at,
        ae.duration_ms,
        ae.error_message
    FROM action_executions ae
    WHERE ae.service_id = p_service_id
      AND ae.action_status = 'FailedActionStatus'
      AND ae.started_at > NOW() - (p_hours || ' hours')::INTERVAL
    ORDER BY ae.started_at DESC;
END;
$$ LANGUAGE plpgsql;

-- Get slow actions (duration > threshold)
CREATE OR REPLACE FUNCTION get_slow_actions(
    p_threshold_ms BIGINT DEFAULT 5000,
    p_hours INTEGER DEFAULT 24
)
RETURNS TABLE (
    operation_id TEXT,
    action_type TEXT,
    object_type TEXT,
    service_id TEXT,
    duration_ms BIGINT,
    started_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        ae.operation_id,
        ae.action_type,
        ae.object_type,
        ae.service_id,
        ae.duration_ms,
        ae.started_at
    FROM action_executions ae
    WHERE ae.duration_ms > p_threshold_ms
      AND ae.started_at > NOW() - (p_hours || ' hours')::INTERVAL
    ORDER BY ae.duration_ms DESC
    LIMIT 50;
END;
$$ LANGUAGE plpgsql;

-- Query by metadata field (e.g., find all migrations for a specific database)
CREATE OR REPLACE FUNCTION query_by_metadata(
    p_action_type TEXT,
    p_object_type TEXT,
    p_metadata_key TEXT,
    p_metadata_value TEXT
)
RETURNS TABLE (
    operation_id TEXT,
    correlation_id TEXT,
    started_at TIMESTAMPTZ,
    action_status TEXT,
    metadata JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        ae.operation_id,
        ae.correlation_id,
        ae.started_at,
        ae.action_status,
        ae.metadata
    FROM action_executions ae
    WHERE ae.action_type = p_action_type
      AND ae.object_type = p_object_type
      AND ae.metadata->>p_metadata_key = p_metadata_value
    ORDER BY ae.started_at DESC;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Retention Policies
-- ============================================================================

-- Keep raw action executions for 90 days
SELECT add_retention_policy('action_executions', INTERVAL '90 days', if_not_exists => TRUE);

-- ============================================================================
-- Sample Queries
-- ============================================================================

-- Uncomment to test:

/*
-- Get all actions in a workflow
SELECT * FROM get_workflow_trace('wf-123');

-- Get failed actions in last 24 hours
SELECT * FROM get_failed_actions('containerservice', 24);

-- Get slow actions
SELECT * FROM get_slow_actions(5000, 24);

-- Find all migrations for a specific database
SELECT * FROM query_by_metadata('TransferAction', 'Database', 'source_database', 'prod-db');

-- Get all container creations
SELECT * FROM action_executions
WHERE action_type = 'CreateAction'
  AND object_type = 'SoftwareApplication'
  AND started_at > NOW() - INTERVAL '24 hours';

-- Get all backups expiring soon
SELECT * FROM action_executions
WHERE action_type = 'UploadAction'
  AND object_type = 'Dataset'
  AND (metadata->>'expires_at')::timestamptz < NOW() + INTERVAL '7 days';
*/

-- ============================================================================
-- GDPR & Compliance Extensions
-- ============================================================================

-- Add GDPR compliance columns to action_executions (if not exists)
DO $$
BEGIN
    -- Data subject tracking
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'action_executions' AND column_name = 'data_subject_id') THEN
        ALTER TABLE action_executions ADD COLUMN data_subject_id TEXT;
    END IF;

    -- Legal basis for processing
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'action_executions' AND column_name = 'legal_basis') THEN
        ALTER TABLE action_executions ADD COLUMN legal_basis TEXT;
    END IF;

    -- Consent tracking
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'action_executions' AND column_name = 'consent_id') THEN
        ALTER TABLE action_executions ADD COLUMN consent_id TEXT;
    END IF;

    -- Data residency/region
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'action_executions' AND column_name = 'data_region') THEN
        ALTER TABLE action_executions ADD COLUMN data_region TEXT DEFAULT 'us';
    END IF;

    -- Retention management
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'action_executions' AND column_name = 'retention_until') THEN
        ALTER TABLE action_executions ADD COLUMN retention_until TIMESTAMPTZ;
    END IF;

    -- PII flags
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'action_executions' AND column_name = 'contains_pii') THEN
        ALTER TABLE action_executions ADD COLUMN contains_pii BOOLEAN DEFAULT FALSE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'action_executions' AND column_name = 'pii_redacted') THEN
        ALTER TABLE action_executions ADD COLUMN pii_redacted BOOLEAN DEFAULT FALSE;
    END IF;
END $$;

-- Index for data subject queries
CREATE INDEX IF NOT EXISTS idx_action_exec_data_subject
    ON action_executions (data_subject_id, started_at DESC)
    WHERE data_subject_id IS NOT NULL;

-- Index for retention management
CREATE INDEX IF NOT EXISTS idx_action_exec_retention
    ON action_executions (retention_until)
    WHERE retention_until IS NOT NULL;

-- Index for regional queries
CREATE INDEX IF NOT EXISTS idx_action_exec_region
    ON action_executions (data_region, started_at DESC);

-- ============================================================================
-- Audit Logging Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS trace_access_audit (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Who accessed
    user_id TEXT NOT NULL,
    user_email TEXT,
    user_ip INET,

    -- What was accessed
    access_type TEXT NOT NULL,  -- 'query', 'view', 'export', 'delete'
    resource_type TEXT NOT NULL,  -- 'workflow_trace', 'action_detail', 'metadata'
    correlation_id TEXT,
    operation_id TEXT,
    data_subject_id TEXT,

    -- How it was accessed
    query_parameters JSONB,
    results_count INTEGER,

    -- Why it was accessed (justification)
    purpose TEXT,
    legal_basis TEXT,

    -- Audit trail
    session_id TEXT,
    request_id TEXT
);

-- Index for audit queries
CREATE INDEX IF NOT EXISTS idx_audit_accessed_at
    ON trace_access_audit (accessed_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_user
    ON trace_access_audit (user_id, accessed_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_correlation
    ON trace_access_audit (correlation_id)
    WHERE correlation_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_audit_data_subject
    ON trace_access_audit (data_subject_id)
    WHERE data_subject_id IS NOT NULL;

-- ============================================================================
-- PII Detection Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS pii_detections (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Where PII was found
    correlation_id TEXT NOT NULL,
    operation_id TEXT NOT NULL,
    location TEXT NOT NULL,  -- 'request', 'response', 'metadata'
    field_path TEXT,  -- JSON path like '$.object.identifier'

    -- What type of PII
    pii_type TEXT NOT NULL,  -- 'email', 'phone', 'ssn', 'credit_card', 'ip_address'
    pattern_matched TEXT,
    confidence FLOAT,  -- 0.0-1.0

    -- Was it redacted/tokenized?
    redacted BOOLEAN DEFAULT FALSE,
    token TEXT,  -- If tokenized

    -- Data subject tracking
    data_subject_id TEXT
);

-- Index for PII queries
CREATE INDEX IF NOT EXISTS idx_pii_correlation
    ON pii_detections (correlation_id, operation_id);

CREATE INDEX IF NOT EXISTS idx_pii_type
    ON pii_detections (pii_type, detected_at DESC);

CREATE INDEX IF NOT EXISTS idx_pii_data_subject
    ON pii_detections (data_subject_id)
    WHERE data_subject_id IS NOT NULL;

-- ============================================================================
-- GDPR Erasure Functions
-- ============================================================================

-- Full erasure: Delete all traces for a data subject or correlation ID
CREATE OR REPLACE FUNCTION gdpr_erase_traces(
    p_data_subject_id TEXT DEFAULT NULL,
    p_correlation_id TEXT DEFAULT NULL,
    p_user_id TEXT DEFAULT 'system',
    p_purpose TEXT DEFAULT 'GDPR Right to Erasure'
)
RETURNS TABLE (
    deleted_actions INTEGER,
    deleted_pii INTEGER,
    s3_urls_to_delete TEXT[]
) AS $$
DECLARE
    v_deleted_actions INTEGER;
    v_deleted_pii INTEGER;
    v_s3_urls TEXT[];
BEGIN
    -- Validate input
    IF p_data_subject_id IS NULL AND p_correlation_id IS NULL THEN
        RAISE EXCEPTION 'Must provide either data_subject_id or correlation_id';
    END IF;

    -- Collect S3 URLs that need to be deleted
    SELECT ARRAY_AGG(DISTINCT url)
    INTO v_s3_urls
    FROM (
        SELECT request_url AS url FROM action_executions
        WHERE (p_data_subject_id IS NULL OR data_subject_id = p_data_subject_id)
          AND (p_correlation_id IS NULL OR correlation_id = p_correlation_id)
          AND request_url IS NOT NULL
          AND request_url NOT LIKE '[REDACTED%'
        UNION
        SELECT response_url AS url FROM action_executions
        WHERE (p_data_subject_id IS NULL OR data_subject_id = p_data_subject_id)
          AND (p_correlation_id IS NULL OR correlation_id = p_correlation_id)
          AND response_url IS NOT NULL
          AND response_url NOT LIKE '[REDACTED%'
        UNION
        SELECT logs_url AS url FROM action_executions
        WHERE (p_data_subject_id IS NULL OR data_subject_id = p_data_subject_id)
          AND (p_correlation_id IS NULL OR correlation_id = p_correlation_id)
          AND logs_url IS NOT NULL
    ) urls;

    -- Log the erasure action in audit table
    INSERT INTO trace_access_audit (
        user_id, access_type, resource_type,
        correlation_id, data_subject_id,
        purpose, legal_basis,
        query_parameters
    ) VALUES (
        p_user_id, 'delete', 'gdpr_erasure',
        p_correlation_id, p_data_subject_id,
        p_purpose, 'GDPR Article 17',
        jsonb_build_object(
            'data_subject_id', p_data_subject_id,
            'correlation_id', p_correlation_id
        )
    );

    -- Delete PII detections
    DELETE FROM pii_detections
    WHERE (p_data_subject_id IS NULL OR data_subject_id = p_data_subject_id)
      AND (p_correlation_id IS NULL OR correlation_id IN (
          SELECT correlation_id FROM action_executions
          WHERE (p_data_subject_id IS NULL OR data_subject_id = p_data_subject_id)
            AND (p_correlation_id IS NULL OR correlation_id = p_correlation_id)
      ));

    GET DIAGNOSTICS v_deleted_pii = ROW_COUNT;

    -- Delete action executions
    DELETE FROM action_executions
    WHERE (p_data_subject_id IS NULL OR data_subject_id = p_data_subject_id)
      AND (p_correlation_id IS NULL OR correlation_id = p_correlation_id);

    GET DIAGNOSTICS v_deleted_actions = ROW_COUNT;

    -- Return results
    RETURN QUERY SELECT v_deleted_actions, v_deleted_pii, v_s3_urls;
END;
$$ LANGUAGE plpgsql;

-- Pseudonymization: Replace identifiable data with hashes
CREATE OR REPLACE FUNCTION gdpr_pseudonymize_traces(
    p_data_subject_id TEXT,
    p_user_id TEXT DEFAULT 'system'
)
RETURNS INTEGER AS $$
DECLARE
    v_updated INTEGER;
BEGIN
    -- Log the pseudonymization
    INSERT INTO trace_access_audit (
        user_id, access_type, resource_type,
        data_subject_id, purpose, legal_basis
    ) VALUES (
        p_user_id, 'pseudonymize', 'gdpr_pseudonymization',
        p_data_subject_id, 'GDPR Article 17 - Pseudonymization', 'GDPR Article 17'
    );

    -- Replace data_subject_id with hash
    UPDATE action_executions
    SET data_subject_id = 'PSEUDONYMIZED-' || MD5(data_subject_id),
        client_ip = NULL,
        user_agent = '[REDACTED]',
        metadata = metadata - 'email' - 'phone' - 'name' - 'address'
    WHERE data_subject_id = p_data_subject_id;

    GET DIAGNOSTICS v_updated = ROW_COUNT;

    RETURN v_updated;
END;
$$ LANGUAGE plpgsql;

-- Export data for GDPR data portability (Article 20)
CREATE OR REPLACE FUNCTION gdpr_export_data(
    p_data_subject_id TEXT
)
RETURNS TABLE (
    correlation_id TEXT,
    operation_id TEXT,
    action_type TEXT,
    started_at TIMESTAMPTZ,
    duration_ms BIGINT,
    action_status TEXT,
    service_id TEXT,
    metadata JSONB,
    request_url TEXT,
    response_url TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        ae.correlation_id,
        ae.operation_id,
        ae.action_type,
        ae.started_at,
        ae.duration_ms,
        ae.action_status,
        ae.service_id,
        ae.metadata,
        ae.request_url,
        ae.response_url
    FROM action_executions ae
    WHERE ae.data_subject_id = p_data_subject_id
    ORDER BY ae.started_at DESC;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Audit Helper Functions
-- ============================================================================

-- Log trace access
CREATE OR REPLACE FUNCTION log_trace_access(
    p_user_id TEXT,
    p_access_type TEXT,
    p_resource_type TEXT,
    p_correlation_id TEXT DEFAULT NULL,
    p_operation_id TEXT DEFAULT NULL,
    p_data_subject_id TEXT DEFAULT NULL,
    p_results_count INTEGER DEFAULT NULL,
    p_purpose TEXT DEFAULT NULL,
    p_query_parameters JSONB DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_audit_id UUID;
BEGIN
    INSERT INTO trace_access_audit (
        user_id, access_type, resource_type,
        correlation_id, operation_id, data_subject_id,
        results_count, purpose, query_parameters
    ) VALUES (
        p_user_id, p_access_type, p_resource_type,
        p_correlation_id, p_operation_id, p_data_subject_id,
        p_results_count, p_purpose, p_query_parameters
    ) RETURNING id INTO v_audit_id;

    RETURN v_audit_id;
END;
$$ LANGUAGE plpgsql;

-- Get audit trail for a data subject
CREATE OR REPLACE FUNCTION get_audit_trail(
    p_data_subject_id TEXT DEFAULT NULL,
    p_correlation_id TEXT DEFAULT NULL,
    p_hours INTEGER DEFAULT 168  -- Default 7 days
)
RETURNS TABLE (
    accessed_at TIMESTAMPTZ,
    user_id TEXT,
    access_type TEXT,
    resource_type TEXT,
    purpose TEXT,
    results_count INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        aa.accessed_at,
        aa.user_id,
        aa.access_type,
        aa.resource_type,
        aa.purpose,
        aa.results_count
    FROM trace_access_audit aa
    WHERE (p_data_subject_id IS NULL OR aa.data_subject_id = p_data_subject_id)
      AND (p_correlation_id IS NULL OR aa.correlation_id = p_correlation_id)
      AND aa.accessed_at > NOW() - (p_hours || ' hours')::INTERVAL
    ORDER BY aa.accessed_at DESC;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Retention Policy Enforcement
-- ============================================================================

-- Auto-delete expired traces
CREATE OR REPLACE FUNCTION delete_expired_traces()
RETURNS INTEGER AS $$
DECLARE
    v_deleted INTEGER;
BEGIN
    DELETE FROM action_executions
    WHERE retention_until IS NOT NULL
      AND retention_until < NOW();

    GET DIAGNOSTICS v_deleted = ROW_COUNT;

    RETURN v_deleted;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions
GRANT ALL ON action_executions TO claude;
GRANT ALL ON action_metadata_schemas TO claude;
GRANT ALL ON trace_access_audit TO claude;
GRANT ALL ON pii_detections TO claude;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO claude;

-- PostgreSQL initialization script for EVE tracing
-- Creates TimescaleDB hypertable for action_executions

-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- Create action_executions table
CREATE TABLE IF NOT EXISTS action_executions (
    -- Primary identifiers
    correlation_id TEXT NOT NULL,
    operation_id TEXT PRIMARY KEY,
    service_id TEXT NOT NULL,

    -- Action details
    action_type TEXT NOT NULL,
    object_type TEXT,
    action_status TEXT NOT NULL, -- 'completed', 'failed', 'active'

    -- Timing
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms DOUBLE PRECISION,

    -- Error handling
    error_message TEXT,
    error_type TEXT,

    -- Payload storage (S3 URLs)
    request_s3_url TEXT,
    response_s3_url TEXT,

    -- Metadata (JSONB for flexible polymorphic data)
    metadata JSONB,

    -- OpenTelemetry integration
    trace_id TEXT,
    span_id TEXT,
    parent_span_id TEXT,

    -- Archival fields
    archived_at TIMESTAMP WITH TIME ZONE,
    archived_s3_key TEXT,

    -- Sampling
    sampled BOOLEAN DEFAULT TRUE,
    sampling_reason TEXT
);

-- Convert to TimescaleDB hypertable (partitioned by time)
SELECT create_hypertable('action_executions', 'started_at',
    if_not_exists => TRUE,
    chunk_time_interval => INTERVAL '1 day'
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_action_executions_correlation_id
    ON action_executions(correlation_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_action_executions_service_id
    ON action_executions(service_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_action_executions_action_type
    ON action_executions(action_type, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_action_executions_status
    ON action_executions(action_status, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_action_executions_trace_id
    ON action_executions(trace_id) WHERE trace_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_action_executions_metadata
    ON action_executions USING GIN(metadata);

CREATE INDEX IF NOT EXISTS idx_action_executions_archived
    ON action_executions(archived_at) WHERE archived_at IS NOT NULL;

-- Helper function: Get all actions for a workflow
CREATE OR REPLACE FUNCTION get_workflow_trace(p_correlation_id TEXT)
RETURNS TABLE (
    operation_id TEXT,
    service_id TEXT,
    action_type TEXT,
    object_type TEXT,
    action_status TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms DOUBLE PRECISION,
    error_message TEXT,
    step_number BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        ae.operation_id,
        ae.service_id,
        ae.action_type,
        ae.object_type,
        ae.action_status,
        ae.started_at,
        ae.completed_at,
        ae.duration_ms,
        ae.error_message,
        ROW_NUMBER() OVER (ORDER BY ae.started_at) as step_number
    FROM action_executions ae
    WHERE ae.correlation_id = p_correlation_id
    ORDER BY ae.started_at;
END;
$$ LANGUAGE plpgsql;

-- Helper function: Get workflow summary statistics
CREATE OR REPLACE FUNCTION get_workflow_stats(p_correlation_id TEXT)
RETURNS TABLE (
    total_steps BIGINT,
    failed_steps BIGINT,
    total_duration_ms DOUBLE PRECISION,
    services_involved TEXT[]
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*) as total_steps,
        COUNT(*) FILTER (WHERE action_status = 'failed') as failed_steps,
        SUM(duration_ms) as total_duration_ms,
        ARRAY_AGG(DISTINCT service_id ORDER BY service_id) as services_involved
    FROM action_executions
    WHERE correlation_id = p_correlation_id;
END;
$$ LANGUAGE plpgsql;

-- Sample data for demonstration
INSERT INTO action_executions (
    correlation_id, operation_id, service_id,
    action_type, object_type, action_status,
    started_at, completed_at, duration_ms,
    metadata, sampled
) VALUES
    -- Workflow 1: Container creation
    ('wf-demo-001', 'op-001', 'containerservice',
     'CreateAction', 'SoftwareApplication', 'completed',
     NOW() - INTERVAL '5 minutes', NOW() - INTERVAL '4 minutes 58 seconds', 2000.0,
     '{"container_name": "nginx-demo", "image": "nginx:latest"}'::jsonb, true),

    ('wf-demo-001', 'op-002', 's3service',
     'UploadAction', 'Dataset', 'completed',
     NOW() - INTERVAL '4 minutes 58 seconds', NOW() - INTERVAL '4 minutes 55 seconds', 3000.0,
     '{"file_name": "config.yml", "size_bytes": 1024}'::jsonb, true),

    ('wf-demo-001', 'op-003', 'workflowstorageservice',
     'CreateAction', 'Workflow', 'completed',
     NOW() - INTERVAL '4 minutes 55 seconds', NOW() - INTERVAL '4 minutes 50 seconds', 5000.0,
     '{"workflow_id": "wf-demo-001", "status": "completed"}'::jsonb, true),

    -- Workflow 2: Failed deployment
    ('wf-demo-002', 'op-004', 'containerservice',
     'CreateAction', 'SoftwareApplication', 'failed',
     NOW() - INTERVAL '3 minutes', NOW() - INTERVAL '2 minutes 55 seconds', 5000.0,
     '{"container_name": "app-demo", "image": "myapp:latest"}'::jsonb, true),

    -- Workflow 3: Slow query
    ('wf-demo-003', 'op-005', 'sparqlservice',
     'SearchAction', 'Dataset', 'completed',
     NOW() - INTERVAL '2 minutes', NOW() - INTERVAL '1 minute 50 seconds', 10000.0,
     '{"query": "SELECT * WHERE { ?s ?p ?o } LIMIT 1000"}'::jsonb, true);

-- Create user for read-only access (e.g., for Grafana)
CREATE USER grafana_reader WITH PASSWORD 'grafana_password';
GRANT CONNECT ON DATABASE eve_traces TO grafana_reader;
GRANT USAGE ON SCHEMA public TO grafana_reader;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO grafana_reader;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO grafana_reader;

-- Output success message
DO $$
BEGIN
    RAISE NOTICE 'EVE tracing database initialized successfully!';
    RAISE NOTICE 'Sample workflows created for demonstration.';
END $$;

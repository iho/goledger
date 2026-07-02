DROP FUNCTION IF EXISTS scrub_audit_log_pii(timestamptz);
DROP FUNCTION IF EXISTS ensure_audit_log_partition(date);

ALTER TABLE audit_logs RENAME TO audit_logs_partitioned;

CREATE TABLE audit_logs (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    request_id VARCHAR(255),
    before_state JSONB,
    after_state JSONB,
    status VARCHAR(20) NOT NULL,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO audit_logs (
    id, user_id, action, resource_type, resource_id, ip_address, user_agent,
    request_id, before_state, after_state, status, error_message, created_at
)
SELECT
    id, user_id, action, resource_type, resource_id, ip_address, user_agent,
    request_id, before_state, after_state, status, error_message, created_at
FROM audit_logs_partitioned;

DROP TABLE audit_logs_partitioned CASCADE;
DROP FUNCTION IF EXISTS reject_audit_mutation();

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_status ON audit_logs(status);
CREATE INDEX idx_audit_logs_user_action ON audit_logs(user_id, action, created_at DESC);

CREATE TRIGGER audit_logs_append_only
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION reject_mutation();

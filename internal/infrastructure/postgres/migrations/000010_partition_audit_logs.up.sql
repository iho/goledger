-- Retention policy support for audit_logs: partition by month so old data
-- can be archived/dropped a whole partition at a time instead of a
-- row-by-row DELETE (which the append-only trigger blocks anyway), and add
-- a controlled bypass so a retention job can scrub PII (ip_address,
-- user_agent) from old rows without opening the table up to general
-- mutation.
--
-- Suggested policy (financial records ~5-7y, personal data minimized
-- sooner): keep full rows for 7 years for compliance/examiners, run
-- scrub_audit_log_pii() after ~90 days to null out ip_address/user_agent,
-- and DROP old monthly partitions once past the 7-year retention window
-- (an operator/ops-job decision - not automated here).

ALTER TABLE audit_logs RENAME TO audit_logs_unpartitioned;

CREATE TABLE audit_logs (
    id VARCHAR(255) NOT NULL,
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Catch-all partition so inserts never fail for a month without an explicit
-- partition; ensure_audit_log_partition() below should be called ahead of
-- time (e.g. monthly cron / CLI command) to keep writes out of default.
CREATE TABLE audit_logs_default PARTITION OF audit_logs DEFAULT;

INSERT INTO audit_logs (
    id, user_id, action, resource_type, resource_id, ip_address, user_agent,
    request_id, before_state, after_state, status, error_message, created_at
)
SELECT
    id, user_id, action, resource_type, resource_id, ip_address, user_agent,
    request_id, before_state, after_state, status, error_message, created_at
FROM audit_logs_unpartitioned;

DROP TABLE audit_logs_unpartitioned;

-- Indexes: created on the partitioned parent, Postgres creates a matching
-- index on every partition (including ones created later).
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_status ON audit_logs(status);
CREATE INDEX idx_audit_logs_user_action ON audit_logs(user_id, action, created_at DESC);

-- reject_audit_mutation replaces the plain reject_mutation() trigger on
-- audit_logs specifically: it still blocks all DELETEs and all UPDATEs
-- except a narrowly-scoped PII scrub, which must opt in per-transaction via
-- `SET LOCAL app.allow_audit_scrub = 'true'`. entries/transfers keep using
-- the unconditional reject_mutation() from migration 000009.
DROP TRIGGER IF EXISTS audit_logs_append_only ON audit_logs;

CREATE OR REPLACE FUNCTION reject_audit_mutation() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND current_setting('app.allow_audit_scrub', true) = 'true' THEN
        RETURN NEW;
    END IF;
    RAISE EXCEPTION 'audit_logs is append-only: % is not permitted (set app.allow_audit_scrub for a PII retention scrub)', TG_OP;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_logs_append_only
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION reject_audit_mutation();

-- ensure_audit_log_partition creates the monthly partition covering
-- for_date if it doesn't already exist. Idempotent; safe to call from a
-- periodic job ahead of the month it covers.
CREATE OR REPLACE FUNCTION ensure_audit_log_partition(for_date date) RETURNS void AS $$
DECLARE
    partition_start date := date_trunc('month', for_date);
    partition_end date := partition_start + interval '1 month';
    partition_name text := 'audit_logs_' || to_char(partition_start, 'YYYY_MM');
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = partition_name) THEN
        EXECUTE format(
            'CREATE TABLE %I PARTITION OF audit_logs FOR VALUES FROM (%L) TO (%L)',
            partition_name, partition_start, partition_end
        );
    END IF;
END;
$$ LANGUAGE plpgsql;

-- scrub_audit_log_pii nulls out ip_address/user_agent on rows older than
-- cutoff, for GDPR-style data minimization ahead of the full 5-7y
-- compliance retention window. Uses the narrow trigger bypass above.
CREATE OR REPLACE FUNCTION scrub_audit_log_pii(cutoff timestamptz) RETURNS bigint AS $$
DECLARE
    affected bigint;
BEGIN
    PERFORM set_config('app.allow_audit_scrub', 'true', true); -- local to this transaction
    UPDATE audit_logs
    SET ip_address = NULL, user_agent = NULL
    WHERE created_at < cutoff AND (ip_address IS NOT NULL OR user_agent IS NOT NULL);
    GET DIAGNOSTICS affected = ROW_COUNT;
    RETURN affected;
END;
$$ LANGUAGE plpgsql;

-- Pre-create partitions for the current month and a year in each direction
-- so existing data (from the migration above) and near-term writes land in
-- a real partition rather than the default one.
DO $$
DECLARE
    m int;
BEGIN
    FOR m IN -12..12 LOOP
        PERFORM ensure_audit_log_partition((date_trunc('month', now()) + (m || ' months')::interval)::date);
    END LOOP;
END $$;

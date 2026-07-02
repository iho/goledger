-- Hash-chain audit_logs for tamper evidence: every row stores the hash of
-- the row before it (globally, across the whole partitioned table) plus a
-- hash of its own content. Altering or deleting a historical row (even via
-- a direct SQL session with elevated privileges bypassing the app) breaks
-- the chain at that point, which verify_audit_log_chain() below detects.
--
-- Writers are serialized through the single-row audit_log_chain_state table
-- (locked FOR UPDATE in the trigger) so the chain has one well-defined
-- order even under concurrent audit log inserts from unrelated
-- transactions. chain_seq (assigned under that same lock) is the
-- authoritative ordering for verification - NOT created_at, which is set by
-- application code before the INSERT and so isn't guaranteed to match true
-- commit/lock-acquisition order under concurrency. This intentionally
-- trades some write concurrency on audit_logs for a verifiable chain -
-- audit log volume is expected to be far below transfer volume, so this
-- should not be a bottleneck.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE audit_logs ADD COLUMN prev_hash CHAR(64);
ALTER TABLE audit_logs ADD COLUMN hash CHAR(64);
ALTER TABLE audit_logs ADD COLUMN chain_seq BIGINT;

CREATE TABLE audit_log_chain_state (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    last_hash CHAR(64) NOT NULL,
    last_seq BIGINT NOT NULL
);
INSERT INTO audit_log_chain_state (id, last_hash, last_seq) VALUES (TRUE, repeat('0', 64), 0);

-- Backfill any rows that existed before this migration, in created_at order
-- (the best available approximation of history - true insertion order
-- isn't recoverable after the fact). Bypasses the append-only trigger the
-- same way the PII scrub job does, scoped to this transaction only.
DO $$
DECLARE
    rec RECORD;
    prior_hash CHAR(64) := repeat('0', 64);
    seq BIGINT := 0;
    canonical TEXT;
    new_hash CHAR(64);
BEGIN
    PERFORM set_config('app.allow_audit_scrub', 'true', true);

    FOR rec IN
        SELECT id, user_id, action, resource_type, resource_id, status, error_message, created_at
        FROM audit_logs ORDER BY created_at, id
    LOOP
        seq := seq + 1;
        canonical := prior_hash || '|' || rec.id || '|' || rec.user_id || '|' || rec.action || '|' ||
                     rec.resource_type || '|' || rec.resource_id || '|' || rec.status || '|' ||
                     COALESCE(rec.error_message, '') || '|' || extract(epoch from rec.created_at)::text;
        new_hash := encode(digest(canonical, 'sha256'), 'hex');

        UPDATE audit_logs SET prev_hash = prior_hash, hash = new_hash, chain_seq = seq WHERE id = rec.id;

        prior_hash := new_hash;
    END LOOP;

    UPDATE audit_log_chain_state SET last_hash = prior_hash, last_seq = seq;
END $$;

CREATE OR REPLACE FUNCTION audit_log_chain_hash() RETURNS TRIGGER AS $$
DECLARE
    prior_hash CHAR(64);
    next_seq BIGINT;
    canonical TEXT;
BEGIN
    SELECT last_hash, last_seq + 1 INTO prior_hash, next_seq FROM audit_log_chain_state FOR UPDATE;

    NEW.prev_hash := prior_hash;
    NEW.chain_seq := next_seq;
    canonical := prior_hash || '|' || NEW.id || '|' || NEW.user_id || '|' || NEW.action || '|' ||
                 NEW.resource_type || '|' || NEW.resource_id || '|' || NEW.status || '|' ||
                 COALESCE(NEW.error_message, '') || '|' || extract(epoch from NEW.created_at)::text;
    NEW.hash := encode(digest(canonical, 'sha256'), 'hex');

    UPDATE audit_log_chain_state SET last_hash = NEW.hash, last_seq = next_seq;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_logs_chain_hash
    BEFORE INSERT ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_log_chain_hash();

-- Unique indexes on a partitioned table must include the partition key
-- (created_at); chain_seq is globally unique by construction (assigned
-- under audit_log_chain_state's lock), so this composite key still catches
-- any accidental duplicate assignment.
CREATE UNIQUE INDEX idx_audit_logs_chain_seq ON audit_logs(chain_seq, created_at);

-- verify_audit_log_chain recomputes every row's hash from its stored
-- content (ordered by chain_seq, the authoritative insertion order) and
-- compares it - and the prev_hash linkage - against what's stored,
-- returning one row per break found. An empty result means the chain is
-- intact. Uses the exact same canonical-string formula and digest call as
-- the trigger above, so there is only one place the hash algorithm is
-- defined.
CREATE OR REPLACE FUNCTION verify_audit_log_chain() RETURNS TABLE(audit_id text, reason text) AS $$
    WITH ordered AS (
        SELECT
            a.id, a.prev_hash, a.hash, a.chain_seq, a.user_id, a.action, a.resource_type,
            a.resource_id, a.status, a.error_message, a.created_at,
            COALESCE(
                LAG(a.hash) OVER (ORDER BY a.chain_seq),
                repeat('0', 64)
            ) AS expected_prev_hash
        FROM audit_logs a
    ),
    checked AS (
        SELECT
            id, prev_hash, hash, expected_prev_hash,
            encode(digest(
                expected_prev_hash || '|' || id || '|' || user_id || '|' || action || '|' ||
                resource_type || '|' || resource_id || '|' || status || '|' ||
                COALESCE(error_message, '') || '|' || extract(epoch from created_at)::text,
                'sha256'
            ), 'hex') AS expected_hash
        FROM ordered
    )
    SELECT id, 'prev_hash does not match previous row''s hash'
    FROM checked WHERE prev_hash IS DISTINCT FROM expected_prev_hash
    UNION ALL
    SELECT id, 'hash does not match recomputed value (row may have been tampered with)'
    FROM checked WHERE hash IS DISTINCT FROM expected_hash;
$$ LANGUAGE sql STABLE;

DROP FUNCTION IF EXISTS verify_audit_log_chain();
DROP INDEX IF EXISTS idx_audit_logs_chain_seq;
DROP TRIGGER IF EXISTS audit_logs_chain_hash ON audit_logs;
DROP FUNCTION IF EXISTS audit_log_chain_hash();
DROP TABLE IF EXISTS audit_log_chain_state;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS chain_seq;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS hash;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS prev_hash;

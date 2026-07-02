DROP TRIGGER IF EXISTS audit_logs_append_only ON audit_logs;
DROP TRIGGER IF EXISTS transfers_append_only ON transfers;
DROP TRIGGER IF EXISTS entries_append_only ON entries;
DROP FUNCTION IF EXISTS reject_mutation();

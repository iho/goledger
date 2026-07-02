-- Enforce append-only semantics at the database level for the tables that
-- make up the audit trail / event log: entries, transfers, and audit_logs.
-- Immutability was previously by application convention only; a bug or a
-- direct SQL session could silently rewrite history. No code path in the
-- application updates or deletes rows in these tables, so this trigger
-- should never fire in normal operation.
CREATE OR REPLACE FUNCTION reject_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION '% is append-only: % is not permitted', TG_TABLE_NAME, TG_OP;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER entries_append_only
    BEFORE UPDATE OR DELETE ON entries
    FOR EACH ROW EXECUTE FUNCTION reject_mutation();

CREATE TRIGGER transfers_append_only
    BEFORE UPDATE OR DELETE ON transfers
    FOR EACH ROW EXECUTE FUNCTION reject_mutation();

CREATE TRIGGER audit_logs_append_only
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION reject_mutation();

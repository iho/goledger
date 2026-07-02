-- Dead-letter support for the outbox: track delivery attempts per event and
-- stop retrying (instead of polling it forever) once a configurable max is
-- exceeded, so one poison message can't block the whole queue behind it.
ALTER TABLE outbox_events ADD COLUMN attempts INT NOT NULL DEFAULT 0;
ALTER TABLE outbox_events ADD COLUMN last_error TEXT;
ALTER TABLE outbox_events ADD COLUMN dead_lettered_at TIMESTAMPTZ;

CREATE INDEX idx_outbox_events_dead_lettered ON outbox_events(dead_lettered_at) WHERE dead_lettered_at IS NOT NULL;

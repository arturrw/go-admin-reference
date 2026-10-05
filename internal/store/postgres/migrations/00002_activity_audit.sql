-- +goose Up
-- The activity feed becomes an audit log: entries are linked to the member who
-- acted and to the record they touched.
ALTER TABLE activity
    ADD COLUMN actor_id  bigint REFERENCES members (id) ON DELETE SET NULL,
    ADD COLUMN entity    text   NOT NULL DEFAULT '',
    ADD COLUMN entity_id bigint NOT NULL DEFAULT 0;
CREATE INDEX activity_at_idx ON activity (at DESC, id DESC);
CREATE INDEX activity_actor_idx ON activity (actor_id, at DESC);

-- +goose Down
DROP INDEX activity_actor_idx;
DROP INDEX activity_at_idx;
ALTER TABLE activity DROP COLUMN entity_id, DROP COLUMN entity, DROP COLUMN actor_id;

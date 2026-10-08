-- +goose Up
-- Invitations: the invitee opens a link with a random token; only its hash is
-- stored, and it expires. Accepting sets the password and clears both columns.
ALTER TABLE members
    ADD COLUMN invite_hash       bytea,
    ADD COLUMN invite_expires_at timestamptz;
CREATE UNIQUE INDEX members_invite_hash_key ON members (invite_hash) WHERE invite_hash IS NOT NULL;

-- +goose Down
DROP INDEX members_invite_hash_key;
ALTER TABLE members DROP COLUMN invite_expires_at, DROP COLUMN invite_hash;
-- +goose Up
-- Which version of the demo dataset a database was seeded with, so a database
-- seeded by an older shape of the demo can be rebuilt.
CREATE TABLE seed_info (
    id      boolean PRIMARY KEY DEFAULT true CHECK (id),
    version integer NOT NULL
);

-- +goose Down
DROP TABLE seed_info;

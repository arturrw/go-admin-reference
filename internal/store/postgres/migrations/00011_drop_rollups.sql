-- +goose Up
-- The dashboard is computed from orders now; the seeded rollups are not read.
DROP TABLE revenue_daily, orders_heatmap;

-- +goose Down
CREATE TABLE revenue_daily (
    day            date   PRIMARY KEY,
    current_cents  bigint NOT NULL,
    previous_cents bigint NOT NULL
);
CREATE TABLE orders_heatmap (
    dow    smallint NOT NULL CHECK (dow BETWEEN 0 AND 6),
    hour   smallint NOT NULL CHECK (hour BETWEEN 0 AND 23),
    orders integer  NOT NULL,
    PRIMARY KEY (dow, hour)
);

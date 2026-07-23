-- Schema for the checkout service (Postgres).
--
-- NOTE: at runtime the schema is created by GORM AutoMigrate from the structs in
-- the model package (see database.newStorage); AutoMigrate is the source of
-- truth. This file mirrors that schema for documentation and for tooling that
-- prefers explicit DDL. Index and constraint names match GORM's defaults.

-- +migrate Up
CREATE TABLE inventory (
    id                 SERIAL PRIMARY KEY,
    sku                TEXT UNIQUE NOT NULL,
    name               TEXT UNIQUE NOT NULL,
    price              NUMERIC(12,2) NOT NULL,
    inventory_quantity INTEGER NOT NULL,
    -- Non-negative stock invariant (model.Item).
    CONSTRAINT chk_inventory_non_negative CHECK (inventory_quantity >= 0)
);

CREATE TABLE orders (
    id          SERIAL PRIMARY KEY,
    reference   TEXT NOT NULL,   -- unique random reference (UUID)
    customer_id TEXT,
    sku_list    TEXT,            -- JSON-encoded array of SKUs
    price       NUMERIC(12,2)
);
CREATE UNIQUE INDEX idx_orders_reference ON orders (reference);

-- Transactional outbox: written in the same tx as the order, drained to the
-- broker by the relay (model.OutboxItem).
CREATE TABLE outbox (
    id            BIGSERIAL PRIMARY KEY,
    event_id      TEXT NOT NULL,   -- business identity; consumer dedup key
    topic         TEXT,            -- broker routing metadata
    partition_key TEXT,            -- broker ordering key
    data          BYTEA,           -- encoded event value, shipped verbatim
    occurred_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ,
    published_at  TIMESTAMPTZ,     -- NULL until relayed to the broker
    delivered_at  TIMESTAMPTZ      -- NULL until downstream delivery (notifier)
);
CREATE UNIQUE INDEX idx_outbox_event_id ON outbox (event_id);
-- Supports the relay's claim scan: WHERE published_at IS NULL.
CREATE INDEX idx_outbox_published_at ON outbox (published_at);

-- +migrate Down
DROP TABLE outbox;
DROP TABLE orders;
DROP TABLE inventory;

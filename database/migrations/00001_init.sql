-- +migrate Up
CREATE TABLE inventory (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    price DOUBLE NOT NULL,
    inventory_quantity INTEGER NOT NULL
);
CREATE TABLE orders (
    id INTEGER PRIMARY KEY,
    reference TEXT UNIQUE, -- Unique random reference
    sku_list TEXT, -- JSON-encoded array of SKUs
    price DOUBLE
);

-- +migrate Down
DROP TABLE orders;
DROP TABLE inventory;
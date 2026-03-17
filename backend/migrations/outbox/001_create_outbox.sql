
CREATE TABLE IF NOT EXISTS outbox (
    id           TEXT        PRIMARY KEY,
    subject      TEXT        NOT NULL,
    payload      BYTEA       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ          -- NULL = pending; set by the relay after NATS ack
);

CREATE INDEX IF NOT EXISTS outbox_pending ON outbox(created_at)
    WHERE published_at IS NULL;

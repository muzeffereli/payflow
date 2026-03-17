
CREATE TABLE IF NOT EXISTS stores (
    id          VARCHAR(36)  PRIMARY KEY,
    owner_id    VARCHAR(36)  NOT NULL UNIQUE,  -- one store per seller
    name        VARCHAR(255) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    email       VARCHAR(255) NOT NULL DEFAULT '',
    commission  INTEGER      NOT NULL DEFAULT 10 CHECK (commission >= 0 AND commission <= 100),
    status      VARCHAR(20)  NOT NULL DEFAULT 'pending'
                             CHECK (status IN ('pending', 'active', 'suspended', 'closed')),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stores_owner_id ON stores (owner_id);
CREATE INDEX IF NOT EXISTS idx_stores_status   ON stores (status);

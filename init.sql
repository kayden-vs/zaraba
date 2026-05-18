-- =============================================================
-- zaraba database initialisation
-- Run once when the PostgreSQL container first starts.
-- =============================================================

-- ── Users ─────────────────────────────────────────────────────
-- Must be created first; wallets and orders foreign-key to this.
CREATE TABLE IF NOT EXISTS users (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    email           VARCHAR(255) NOT NULL,
    hashed_password BYTEA        NOT NULL,
    created         TIMESTAMP    NOT NULL DEFAULT NOW(),

    CONSTRAINT users_uc_email UNIQUE (email)
);

-- ── Sessions (scs/postgresstore) ──────────────────────────────
-- Required by github.com/alexedwards/scs/postgresstore.
-- The library expects exactly this schema.
CREATE TABLE IF NOT EXISTS sessions (
    token  TEXT PRIMARY KEY,
    data   BYTEA       NOT NULL,
    expiry TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions (expiry);

-- ── Wallets ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS wallets (
    user_id    BIGINT    PRIMARY KEY,
    balance    BIGINT    NOT NULL DEFAULT 0,
    locked     BIGINT    NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_wallet_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

-- ── Orders ────────────────────────────────────────────────────
-- The application also calls EnsureSchema() at start-up which
-- is idempotent, so this just pre-creates the table.
CREATE TABLE IF NOT EXISTS orders (
    id               BIGSERIAL    PRIMARY KEY,
    user_id          BIGINT       NOT NULL,
    engine_order_id  BIGINT,
    symbol           VARCHAR(20)  NOT NULL,
    side             VARCHAR(4)   NOT NULL,
    order_type       VARCHAR(16)  NOT NULL DEFAULT 'LIMIT',
    price            BIGINT       NOT NULL,
    quantity         BIGINT       NOT NULL,
    filled           BIGINT       NOT NULL DEFAULT 0,
    filled_notional  BIGINT       NOT NULL DEFAULT 0,
    avg_fill_price   BIGINT       NOT NULL DEFAULT 0,
    status           VARCHAR(20)  NOT NULL,
    created_at       TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP    NOT NULL DEFAULT NOW(),
    closed_at        TIMESTAMP,

    CONSTRAINT fk_orders_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id
    ON orders(user_id);

CREATE INDEX IF NOT EXISTS idx_orders_symbol_status
    ON orders(symbol, status);

CREATE INDEX IF NOT EXISTS idx_orders_user_status_created
    ON orders(user_id, status, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_engine_order_id
    ON orders(engine_order_id)
    WHERE engine_order_id IS NOT NULL;

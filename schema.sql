-- Wallets table
-- One wallet per user (USDT only)

CREATE TABLE IF NOT EXISTS wallets (
    user_id BIGINT PRIMARY KEY,
    balance BIGINT NOT NULL DEFAULT 0, -- total balance (micro-USDT)
    locked  BIGINT NOT NULL DEFAULT 0, -- funds reserved for open orders
    updated_at TIMESTAMP NOT NULL DEFAULT now(),

    CONSTRAINT fk_wallet_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

-- Orders table
-- Stores full order history

CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    engine_order_id BIGINT,
    symbol VARCHAR(20) NOT NULL,   -- BTCUSDT, ETHUSDT
    side VARCHAR(4) NOT NULL,      -- BUY / SELL
    order_type VARCHAR(16) NOT NULL DEFAULT 'LIMIT', -- LIMIT / MARKET
    price BIGINT NOT NULL,         -- price in micro-USDT
    quantity BIGINT NOT NULL,      -- virtual base quantity
    filled BIGINT NOT NULL DEFAULT 0,
    filled_notional BIGINT NOT NULL DEFAULT 0, -- micro-USDT filled value
    avg_fill_price BIGINT NOT NULL DEFAULT 0,  -- micro-USDT average fill
    status VARCHAR(20) NOT NULL,   -- OPEN / FILLED / CANCELLED
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    closed_at TIMESTAMP,

    CONSTRAINT fk_orders_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

-- Indexes (performance)

CREATE INDEX IF NOT EXISTS idx_orders_user_id
    ON orders(user_id);

CREATE INDEX IF NOT EXISTS idx_orders_symbol_status
    ON orders(symbol, status);

CREATE INDEX IF NOT EXISTS idx_orders_user_status_created
    ON orders(user_id, status, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_engine_order_id
    ON orders(engine_order_id)
    WHERE engine_order_id IS NOT NULL;

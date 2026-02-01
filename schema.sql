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
    symbol VARCHAR(20) NOT NULL,   -- BTCUSDT, ETHUSDT
    side VARCHAR(4) NOT NULL,      -- BUY / SELL
    price BIGINT NOT NULL,         -- price in micro-USDT
    quantity BIGINT NOT NULL,      -- virtual base quantity
    filled BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL,   -- OPEN / FILLED / CANCELLED
    created_at TIMESTAMP NOT NULL DEFAULT now(),

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

CREATE TABLE IF NOT EXISTS transactions
(
    id                INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tx_hash           TEXT NOT NULL,
    user_wallet       TEXT NOT NULL,
    merchant_wallet   TEXT NOT NULL,
    price             NUMERIC(18, 10) NOT NULL,
    type              TEXT NOT NULL,
    payload           JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transactions_tx_hash
    ON transactions (tx_hash);

CREATE INDEX IF NOT EXISTS idx_transactions_user_wallet
    ON transactions (user_wallet);

CREATE INDEX IF NOT EXISTS idx_transactions_merchant_wallet
    ON transactions (merchant_wallet);

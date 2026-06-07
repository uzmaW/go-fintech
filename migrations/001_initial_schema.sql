-- Migration: 001_initial_schema.sql
-- Create all core tables for the transaction processing system

-- ============================================================================
-- ACCOUNTS TABLE (Hash partitioned by account_id for horizontal scaling)
-- ============================================================================
CREATE TABLE accounts (
    account_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id             UUID NOT NULL,
    account_number          VARCHAR(20) UNIQUE NOT NULL,
    account_type            VARCHAR(20) NOT NULL CHECK (account_type IN ('CHECKING', 'SAVINGS', 'CREDIT')),
    currency                VARCHAR(3) NOT NULL DEFAULT 'USD',
    status                  VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'frozen', 'closed')),
    balance_available       NUMERIC(19, 4) NOT NULL DEFAULT 0,
    balance_current         NUMERIC(19, 4) NOT NULL DEFAULT 0,
    overdraft_limit         NUMERIC(19, 4) NOT NULL DEFAULT 0,
    daily_withdrawal_limit  NUMERIC(19, 4),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata                JSONB,

    CONSTRAINT chk_balance_non_negative CHECK (balance_current >= -overdraft_limit)
) PARTITION BY HASH (account_id);

-- Create 16 partitions for horizontal scaling
CREATE TABLE accounts_p0  PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 0);
CREATE TABLE accounts_p1  PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 1);
CREATE TABLE accounts_p2  PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 2);
CREATE TABLE accounts_p3  PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 3);
CREATE TABLE accounts_p4  PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 4);
CREATE TABLE accounts_p5  PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 5);
CREATE TABLE accounts_p6  PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 6);
CREATE TABLE accounts_p7  PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 7);
CREATE TABLE accounts_p8  PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 8);
CREATE TABLE accounts_p9  PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 9);
CREATE TABLE accounts_p10 PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 10);
CREATE TABLE accounts_p11 PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 11);
CREATE TABLE accounts_p12 PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 12);
CREATE TABLE accounts_p13 PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 13);
CREATE TABLE accounts_p14 PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 14);
CREATE TABLE accounts_p15 PARTITION OF accounts FOR VALUES WITH (MODULUS 16, REMAINDER 15);

CREATE INDEX idx_accounts_customer ON accounts (customer_id);
CREATE INDEX idx_accounts_number ON accounts (account_number);
CREATE INDEX idx_accounts_status ON accounts (status);

-- ============================================================================
-- TRANSACTIONS TABLE (Range partitioned by created_at monthly)
-- ============================================================================
CREATE TABLE transactions (
    transaction_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key     VARCHAR(64) UNIQUE NOT NULL,
    transaction_type    VARCHAR(30) NOT NULL CHECK (transaction_type IN ('DEPOSIT', 'WITHDRAWAL', 'TRANSFER', 'CARD_AUTH', 'ACH_DEBIT', 'ACH_CREDIT')),
    status              VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'AUTHORIZED', 'SETTLED', 'FAILED', 'REVERSED')),

    source_account_id   UUID,
    target_account_id   UUID,
    merchant_id         UUID,

    amount              NUMERIC(19, 4) NOT NULL,
    currency            VARCHAR(3) NOT NULL,
    fee_amount          NUMERIC(19, 4) DEFAULT 0,
    fx_rate             NUMERIC(19, 8),

    fraud_score         NUMERIC(5, 2),
    risk_level          VARCHAR(10),
    mfa_verified        BOOLEAN DEFAULT false,

    card_id             UUID,
    card_last4          VARCHAR(4),
    merchant_category   VARCHAR(4),
    authorization_code  VARCHAR(10),

    initiated_by        VARCHAR(50) DEFAULT 'USER',
    initiated_at        TIMESTAMPTZ NOT NULL,
    authorized_at       TIMESTAMPTZ,
    settled_at          TIMESTAMPTZ,
    reversed_at         TIMESTAMPTZ,

    ip_address          INET,
    user_agent          TEXT,
    device_fingerprint  VARCHAR(64),
    metadata            JSONB,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY RANGE (created_at);

-- Create monthly partitions
CREATE TABLE transactions_2025_11 PARTITION OF transactions
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
CREATE TABLE transactions_2025_12 PARTITION OF transactions
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');
CREATE TABLE transactions_2026_01 PARTITION OF transactions
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE transactions_2026_02 PARTITION OF transactions
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE transactions_2026_03 PARTITION OF transactions
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE transactions_2026_04 PARTITION OF transactions
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE INDEX idx_tx_source_account ON transactions (source_account_id, created_at DESC);
CREATE INDEX idx_tx_target_account ON transactions (target_account_id, created_at DESC);
CREATE INDEX idx_tx_status ON transactions (status, created_at) WHERE status IN ('PENDING', 'AUTHORIZED');
CREATE INDEX idx_tx_idempotency ON transactions (idempotency_key);
CREATE INDEX idx_tx_created_at ON transactions (created_at DESC);

-- ============================================================================
-- LEDGER ENTRIES TABLE (Immutable, Double-Entry, Range partitioned by posted_at)
-- ============================================================================
CREATE TABLE ledger_entries (
    entry_id            BIGSERIAL,
    transaction_id      UUID NOT NULL,
    account_id          UUID NOT NULL,
    entry_type          VARCHAR(10) NOT NULL CHECK (entry_type IN ('DEBIT', 'CREDIT')),
    amount              NUMERIC(19, 4) NOT NULL,
    currency            VARCHAR(3) NOT NULL,
    balance_after       NUMERIC(19, 4) NOT NULL,
    description         TEXT,
    reference_id        VARCHAR(100),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    posted_at           TIMESTAMPTZ NOT NULL,
    idempotency_key     VARCHAR(64) UNIQUE,
    metadata            JSONB,

    PRIMARY KEY (entry_id, posted_at)
) PARTITION BY RANGE (posted_at);

-- Monthly partitions
CREATE TABLE ledger_entries_2025_11 PARTITION OF ledger_entries
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
CREATE TABLE ledger_entries_2025_12 PARTITION OF ledger_entries
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');
CREATE TABLE ledger_entries_2026_01 PARTITION OF ledger_entries
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE ledger_entries_2026_02 PARTITION OF ledger_entries
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE ledger_entries_2026_03 PARTITION OF ledger_entries
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE ledger_entries_2026_04 PARTITION OF ledger_entries
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE INDEX idx_ledger_tx ON ledger_entries (transaction_id);
CREATE INDEX idx_ledger_account_time ON ledger_entries (account_id, posted_at DESC);
CREATE INDEX idx_ledger_posted_at ON ledger_entries (posted_at DESC);

-- Constraint to enforce double-entry bookkeeping
ALTER TABLE ledger_entries ADD CONSTRAINT chk_double_entry
    CHECK (
        (entry_type = 'DEBIT' AND amount > 0) OR
        (entry_type = 'CREDIT' AND amount > 0)
    );

-- ============================================================================
-- CARDS TABLE
-- ============================================================================
CREATE TABLE cards (
    card_id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id          UUID NOT NULL REFERENCES accounts(account_id),
    card_number_hash    VARCHAR(64) UNIQUE NOT NULL,
    card_last4          VARCHAR(4) NOT NULL,
    card_type           VARCHAR(10) NOT NULL CHECK (card_type IN ('DEBIT', 'CREDIT', 'PREPAID')),
    network             VARCHAR(10) NOT NULL CHECK (network IN ('VISA', 'MASTERCARD', 'AMEX')),
    status              VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'blocked', 'expired')),
    daily_limit         NUMERIC(19, 4),
    expiry_month        SMALLINT NOT NULL CHECK (expiry_month BETWEEN 1 AND 12),
    expiry_year         SMALLINT NOT NULL,
    cvv_hash            VARCHAR(64),
    pin_hash            VARCHAR(64),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    activated_at        TIMESTAMPTZ,
    blocked_at          TIMESTAMPTZ
);

CREATE INDEX idx_cards_account ON cards (account_id);
CREATE INDEX idx_cards_status ON cards (status);
CREATE INDEX idx_cards_number_hash ON cards (card_number_hash);

-- ============================================================================
-- HOLDS TABLE (For pending authorizations)
-- ============================================================================
CREATE TABLE holds (
    hold_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id      UUID NOT NULL REFERENCES accounts(account_id),
    transaction_id  UUID NOT NULL REFERENCES transactions(transaction_id),
    amount          NUMERIC(19, 4) NOT NULL,
    currency        VARCHAR(3) NOT NULL DEFAULT 'USD',
    status          VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'released', 'settled', 'expired')),
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_holds_account ON holds (account_id);
CREATE INDEX idx_holds_transaction ON holds (transaction_id);
CREATE INDEX idx_holds_status ON holds (status) WHERE status = 'active';
CREATE INDEX idx_holds_expires ON holds (expires_at) WHERE status = 'active';

-- ============================================================================
-- AUDIT LOG TABLE (Immutable, Range partitioned weekly)
-- ============================================================================
CREATE TABLE audit_log (
    audit_id           BIGSERIAL PRIMARY KEY,
    event_type         VARCHAR(50) NOT NULL,
    actor_id           UUID,
    actor_type         VARCHAR(20) CHECK (actor_type IN ('USER', 'SERVICE', 'ADMIN')),
    resource_type      VARCHAR(50),
    resource_id        UUID,
    action             VARCHAR(50) NOT NULL,
    status             VARCHAR(20),
    ip_address         INET,
    user_agent         TEXT,
    request_payload    JSONB,
    response_payload   JSONB,
    error_message      TEXT,
    timestamp          TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY RANGE (timestamp);

-- Weekly partitions
CREATE TABLE audit_log_2025_w46 PARTITION OF audit_log
    FOR VALUES FROM ('2025-11-10') TO ('2025-11-17');
CREATE TABLE audit_log_2025_w47 PARTITION OF audit_log
    FOR VALUES FROM ('2025-11-17') TO ('2025-11-24');
CREATE TABLE audit_log_2025_w48 PARTITION OF audit_log
    FOR VALUES FROM ('2025-11-24') TO ('2025-12-01');
CREATE TABLE audit_log_2025_w49 PARTITION OF audit_log
    FOR VALUES FROM ('2025-12-01') TO ('2025-12-08');
CREATE TABLE audit_log_2025_w50 PARTITION OF audit_log
    FOR VALUES FROM ('2025-12-08') TO ('2025-12-15');
CREATE TABLE audit_log_2025_w51 PARTITION OF audit_log
    FOR VALUES FROM ('2025-12-15') TO ('2025-12-22');
CREATE TABLE audit_log_2025_w52 PARTITION OF audit_log
    FOR VALUES FROM ('2025-12-22') TO ('2025-12-29');
CREATE TABLE audit_log_2026_w01 PARTITION OF audit_log
    FOR VALUES FROM ('2025-12-29') TO ('2026-01-05');
CREATE TABLE audit_log_2026_w02 PARTITION OF audit_log
    FOR VALUES FROM ('2026-01-05') TO ('2026-01-12');
CREATE TABLE audit_log_2026_w03 PARTITION OF audit_log
    FOR VALUES FROM ('2026-01-12') TO ('2026-01-19');
CREATE TABLE audit_log_2026_w04 PARTITION OF audit_log
    FOR VALUES FROM ('2026-01-19') TO ('2026-01-26');
CREATE TABLE audit_log_2026_w05 PARTITION OF audit_log
    FOR VALUES FROM ('2026-01-26') TO ('2026-02-02');
CREATE TABLE audit_log_2026_w06 PARTITION OF audit_log
    FOR VALUES FROM ('2026-02-02') TO ('2026-02-09');
CREATE TABLE audit_log_2026_w07 PARTITION OF audit_log
    FOR VALUES FROM ('2026-02-09') TO ('2026-02-16');
CREATE TABLE audit_log_2026_w08 PARTITION OF audit_log
    FOR VALUES FROM ('2026-02-16') TO ('2026-02-23');
CREATE TABLE audit_log_2026_w09 PARTITION OF audit_log
    FOR VALUES FROM ('2026-02-23') TO ('2026-03-02');
CREATE TABLE audit_log_2026_w10 PARTITION OF audit_log
    FOR VALUES FROM ('2026-03-02') TO ('2026-03-09');
CREATE TABLE audit_log_2026_w11 PARTITION OF audit_log
    FOR VALUES FROM ('2026-03-09') TO ('2026-03-16');
CREATE TABLE audit_log_2026_w12 PARTITION OF audit_log
    FOR VALUES FROM ('2026-03-16') TO ('2026-03-23');
CREATE TABLE audit_log_2026_w13 PARTITION OF audit_log
    FOR VALUES FROM ('2026-03-23') TO ('2026-03-30');
CREATE TABLE audit_log_2026_w14 PARTITION OF audit_log
    FOR VALUES FROM ('2026-03-30') TO ('2026-04-06');
CREATE TABLE audit_log_2026_w15 PARTITION OF audit_log
    FOR VALUES FROM ('2026-04-06') TO ('2026-04-13');
CREATE TABLE audit_log_2026_w16 PARTITION OF audit_log
    FOR VALUES FROM ('2026-04-13') TO ('2026-04-20');
CREATE TABLE audit_log_2026_w17 PARTITION OF audit_log
    FOR VALUES FROM ('2026-04-20') TO ('2026-04-27');

CREATE INDEX idx_audit_actor ON audit_log (actor_id, timestamp DESC);
CREATE INDEX idx_audit_resource ON audit_log (resource_type, resource_id, timestamp DESC);
CREATE INDEX idx_audit_event_type ON audit_log (event_type, timestamp DESC);

-- ============================================================================
-- RECONCILIATION SUMMARY TABLE
-- ============================================================================
CREATE TABLE reconciliation_summary (
    id                  SERIAL PRIMARY KEY,
    date                DATE NOT NULL,
    internal_count      INTEGER,
    internal_amount     NUMERIC(19, 4),
    external_count      INTEGER,
    external_amount    NUMERIC(19, 4),
    matched_count      INTEGER,
    matched_amount     NUMERIC(19, 4),
    discrepancy_count   INTEGER,
    mismatch_count     INTEGER,
    status              VARCHAR(20) DEFAULT 'PENDING',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_recon_date ON reconciliation_summary (date);
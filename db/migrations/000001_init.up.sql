CREATE TABLE IF NOT EXISTS accounts (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    document_number VARCHAR(20) NOT NULL UNIQUE,
    created_on      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS operation_types (
    id          INTEGER     PRIMARY KEY,
    description VARCHAR(50) NOT NULL,
    created_on  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transactions (
    id                UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id        UUID          NOT NULL REFERENCES accounts(id),
    operation_type_id INTEGER       NOT NULL REFERENCES operation_types(id),
    amount            NUMERIC(15,2) NOT NULL,
    event_date        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_on        TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS installment_plans (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id     UUID        NOT NULL UNIQUE REFERENCES transactions(id),
    total_installments INTEGER     NOT NULL CHECK (total_installments > 0),
    created_on         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS installment_schedules (
    id                  UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    installment_plan_id UUID          NOT NULL REFERENCES installment_plans(id),
    installment_no      INTEGER       NOT NULL CHECK (installment_no > 0),
    amount              NUMERIC(15,2) NOT NULL,
    due_date            DATE          NOT NULL,
    created_on          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (installment_plan_id, installment_no)
);

INSERT INTO operation_types (id, description) VALUES
    (1, 'Normal Purchase'),
    (2, 'Purchase with Installments'),
    (3, 'Withdrawal'),
    (4, 'Credit Voucher')
ON CONFLICT (id) DO NOTHING;

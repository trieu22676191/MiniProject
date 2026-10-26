-- +goose Up
CREATE TABLE mini_order.orders (
    id uuid PRIMARY KEY,
    tenant_id uuid NOT NULL,
    code text NOT NULL,
    status text NOT NULL CHECK (status IN ('PENDING_APPROVAL', 'APPROVED', 'RESERVED', 'OUT_OF_STOCK', 'REJECTED')),
    created_by text NOT NULL,
    decided_by text,
    reject_reason text,
    total_amount numeric(18,2) NOT NULL CHECK (total_amount >= 0),
    currency_code char(3) NOT NULL DEFAULT 'VND' CHECK (currency_code = 'VND'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, code),
    UNIQUE (tenant_id, id)
);

CREATE TABLE mini_order.order_lines (
    id uuid PRIMARY KEY,
    tenant_id uuid NOT NULL,
    order_id uuid NOT NULL,
    sku text NOT NULL,
    quantity bigint NOT NULL CHECK (quantity > 0),
    unit_price numeric(18,2) NOT NULL CHECK (unit_price >= 0),
    line_amount numeric(18,2) NOT NULL CHECK (line_amount >= 0),
    UNIQUE (tenant_id, order_id, sku),
    FOREIGN KEY (tenant_id, order_id) REFERENCES mini_order.orders(tenant_id, id)
);

CREATE TABLE mini_order.outbox (
    id uuid PRIMARY KEY,
    tenant_id uuid NOT NULL,
    type text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE TABLE mini_order.inbox (
    tenant_id uuid NOT NULL,
    consumer text NOT NULL,
    event_id uuid NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, consumer, event_id)
);

CREATE INDEX orders_tenant_created_idx ON mini_order.orders (tenant_id, created_at DESC);
CREATE INDEX order_lines_tenant_order_idx ON mini_order.order_lines (tenant_id, order_id);
CREATE INDEX order_outbox_pending_idx ON mini_order.outbox (id) WHERE published_at IS NULL;

ALTER TABLE mini_order.orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_order.orders FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_order.orders
  USING (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE mini_order.order_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_order.order_lines FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_order.order_lines
  USING (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE mini_order.outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_order.outbox FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_order.outbox
  USING (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE mini_order.inbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_order.inbox FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_order.inbox
  USING (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

GRANT SELECT, UPDATE ON mini_order.outbox TO mini_relay;

-- +goose Down
DROP TABLE mini_order.inbox;
DROP TABLE mini_order.outbox;
DROP TABLE mini_order.order_lines;
DROP TABLE mini_order.orders;

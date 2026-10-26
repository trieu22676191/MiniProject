-- +goose Up
CREATE TABLE mini_stock.stock_items (
    tenant_id uuid NOT NULL,
    sku text NOT NULL,
    name text NOT NULL,
    unit_price numeric(18,2) NOT NULL CHECK (unit_price >= 0),
    currency_code char(3) NOT NULL DEFAULT 'VND' CHECK (currency_code = 'VND'),
    on_hand bigint NOT NULL CHECK (on_hand >= 0),
    reserved bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, sku),
    CHECK (reserved >= 0 AND on_hand >= reserved)
);

CREATE TABLE mini_stock.reservations (
    id uuid PRIMARY KEY,
    tenant_id uuid NOT NULL,
    order_id uuid NOT NULL,
    status text NOT NULL CHECK (status = 'HELD'),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, order_id),
    UNIQUE (tenant_id, id)
);

CREATE TABLE mini_stock.reservation_lines (
    tenant_id uuid NOT NULL,
    reservation_id uuid NOT NULL,
    sku text NOT NULL,
    quantity bigint NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (tenant_id, reservation_id, sku),
    FOREIGN KEY (tenant_id, reservation_id) REFERENCES mini_stock.reservations(tenant_id, id)
);

CREATE TABLE mini_stock.outbox (
    id uuid PRIMARY KEY,
    tenant_id uuid NOT NULL,
    type text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE TABLE mini_stock.inbox (
    tenant_id uuid NOT NULL,
    consumer text NOT NULL,
    event_id uuid NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, consumer, event_id)
);

CREATE INDEX stock_reservations_tenant_order_idx ON mini_stock.reservations (tenant_id, order_id);
CREATE INDEX stock_outbox_pending_idx ON mini_stock.outbox (id) WHERE published_at IS NULL;

ALTER TABLE mini_stock.stock_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_stock.stock_items FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_stock.stock_items
  USING (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE mini_stock.reservations ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_stock.reservations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_stock.reservations
  USING (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE mini_stock.reservation_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_stock.reservation_lines FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_stock.reservation_lines
  USING (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE mini_stock.outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_stock.outbox FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_stock.outbox
  USING (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE mini_stock.inbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE mini_stock.inbox FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON mini_stock.inbox
  USING (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

GRANT SELECT, UPDATE ON mini_stock.outbox TO mini_relay;

-- +goose Down
DROP TABLE mini_stock.inbox;
DROP TABLE mini_stock.outbox;
DROP TABLE mini_stock.reservation_lines;
DROP TABLE mini_stock.reservations;
DROP TABLE mini_stock.stock_items;

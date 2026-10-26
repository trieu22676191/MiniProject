-- name: GetOrderByID :one
SELECT id, code, status, created_by, decided_by, reject_reason,
       total_amount::text AS total_amount, currency_code
FROM mini_order.orders
WHERE id = $1;

-- name: ListOrders :many
SELECT id, code, status, created_by, decided_by, reject_reason,
       total_amount::text AS total_amount, currency_code
FROM mini_order.orders
ORDER BY created_at DESC, id DESC
LIMIT $1;

-- name: LockOrder :one
SELECT id, code, status, created_by, decided_by, reject_reason,
       total_amount::text AS total_amount, currency_code
FROM mini_order.orders
WHERE id = $1
FOR UPDATE;

-- name: NextOrderCode :one
SELECT nextval('mini_order.order_code_seq')::bigint AS code_number;

-- name: InsertOrder :exec
INSERT INTO mini_order.orders
  (id, tenant_id, code, status, created_by, total_amount, currency_code)
VALUES ($1, $2, $3, $4, $5, $6::text::numeric, $7);

-- name: InsertOrderLine :exec
INSERT INTO mini_order.order_lines
  (id, tenant_id, order_id, sku, quantity, unit_price, line_amount)
VALUES ($1, $2, $3, $4, $5, $6::text::numeric, $7::text::numeric);

-- name: UpdateOrderDecision :exec
UPDATE mini_order.orders
SET status = $2, decided_by = $3, reject_reason = $4, updated_at = now()
WHERE id = $1;

-- name: SetOrderStatus :exec
UPDATE mini_order.orders
SET status = $2, updated_at = now()
WHERE id = $1;

-- name: InsertOrderOutbox :exec
INSERT INTO mini_order.outbox (id, tenant_id, type, payload)
VALUES ($1, $2, $3, $4::jsonb);

-- name: InsertInbox :execrows
INSERT INTO mini_order.inbox (tenant_id, consumer, event_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: ApplyReservationResult :execrows
UPDATE mini_order.orders
SET status = $2, updated_at = now()
WHERE id = $1 AND status = 'APPROVED';

-- name: ListOrderLines :many
SELECT sku, quantity, unit_price::text AS unit_price,
       line_amount::text AS line_amount
FROM mini_order.order_lines
WHERE order_id = $1
ORDER BY sku;

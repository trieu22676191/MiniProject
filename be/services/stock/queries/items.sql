-- name: ListStockItems :many
SELECT sku, name, unit_price::text AS unit_price, currency_code,
       on_hand, reserved
FROM mini_stock.stock_items
ORDER BY sku;

-- name: GetStockItems :many
SELECT sku, name, unit_price::text AS unit_price, currency_code,
       on_hand, reserved
FROM mini_stock.stock_items
WHERE sku = ANY(sqlc.arg(skus)::text[])
ORDER BY sku;

-- name: InsertInbox :execrows
INSERT INTO mini_stock.inbox (tenant_id, consumer, event_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: LockStockForReservation :many
SELECT sku, on_hand, reserved
FROM mini_stock.stock_items
WHERE sku = ANY(sqlc.arg(skus)::text[])
ORDER BY sku
FOR UPDATE;

-- name: IncreaseReserved :exec
UPDATE mini_stock.stock_items
SET reserved = reserved + $3
WHERE tenant_id = $1 AND sku = $2;

-- name: InsertReservation :exec
INSERT INTO mini_stock.reservations (id, tenant_id, order_id, status)
VALUES ($1, $2, $3, 'HELD');

-- name: InsertReservationLine :exec
INSERT INTO mini_stock.reservation_lines (tenant_id, reservation_id, sku, quantity)
VALUES ($1, $2, $3, $4);

-- name: InsertStockOutbox :exec
INSERT INTO mini_stock.outbox (id, tenant_id, type, payload)
VALUES ($1, $2, $3, $4::jsonb);

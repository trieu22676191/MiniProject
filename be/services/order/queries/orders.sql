-- name: GetOrderByID :one
SELECT id, code, status, created_by, decided_by, reject_reason,
       total_amount::text AS total_amount, currency_code
FROM mini_order.orders
WHERE id = $1;

-- name: ListOrderLines :many
SELECT sku, quantity, unit_price::text AS unit_price,
       line_amount::text AS line_amount
FROM mini_order.order_lines
WHERE order_id = $1
ORDER BY sku;

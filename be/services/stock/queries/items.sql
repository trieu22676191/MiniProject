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

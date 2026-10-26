-- +goose Up
SELECT set_config('app.tenant_id', '11111111-1111-1111-1111-111111111111', true);
INSERT INTO mini_stock.stock_items
  (tenant_id, sku, name, unit_price, currency_code, on_hand, reserved)
VALUES
  ('11111111-1111-1111-1111-111111111111', 'BAN-01', 'Bàn làm việc', 1250000.00, 'VND', 10, 0),
  ('11111111-1111-1111-1111-111111111111', 'GHE-01', 'Ghế văn phòng', 550000.00, 'VND', 8, 0),
  ('11111111-1111-1111-1111-111111111111', 'DEN-01', 'Đèn bàn', 180000.00, 'VND', 1, 0);

SELECT set_config('app.tenant_id', '22222222-2222-2222-2222-222222222222', true);
INSERT INTO mini_stock.stock_items
  (tenant_id, sku, name, unit_price, currency_code, on_hand, reserved)
VALUES
  ('22222222-2222-2222-2222-222222222222', 'BAN-01', 'Bàn làm việc B', 1350000.00, 'VND', 5, 0),
  ('22222222-2222-2222-2222-222222222222', 'GHE-01', 'Ghế văn phòng B', 600000.00, 'VND', 4, 0);

-- +goose Down
SELECT set_config('app.tenant_id', '11111111-1111-1111-1111-111111111111', true);
DELETE FROM mini_stock.stock_items WHERE sku IN ('BAN-01', 'GHE-01', 'DEN-01');
SELECT set_config('app.tenant_id', '22222222-2222-2222-2222-222222222222', true);
DELETE FROM mini_stock.stock_items WHERE sku IN ('BAN-01', 'GHE-01');

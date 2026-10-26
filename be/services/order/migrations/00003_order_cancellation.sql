-- +goose Up
ALTER TABLE mini_order.orders DROP CONSTRAINT orders_status_check;
ALTER TABLE mini_order.orders ADD CONSTRAINT orders_status_check
  CHECK (status IN ('PENDING_APPROVAL', 'APPROVED', 'RESERVED', 'OUT_OF_STOCK', 'REJECTED', 'CANCELLED'));

-- +goose Down
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM mini_order.orders WHERE status = 'CANCELLED') THEN
    RAISE EXCEPTION 'cannot roll back order cancellation while cancelled orders exist';
  END IF;
END $$;
ALTER TABLE mini_order.orders DROP CONSTRAINT orders_status_check;
ALTER TABLE mini_order.orders ADD CONSTRAINT orders_status_check
  CHECK (status IN ('PENDING_APPROVAL', 'APPROVED', 'RESERVED', 'OUT_OF_STOCK', 'REJECTED'));

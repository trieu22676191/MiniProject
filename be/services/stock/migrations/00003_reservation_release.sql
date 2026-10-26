-- +goose Up
ALTER TABLE mini_stock.reservations DROP CONSTRAINT reservations_status_check;
ALTER TABLE mini_stock.reservations ADD CONSTRAINT reservations_status_check
  CHECK (status IN ('HELD', 'RELEASED'));

-- +goose Down
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM mini_stock.reservations WHERE status = 'RELEASED') THEN
    RAISE EXCEPTION 'cannot roll back reservation release while released reservations exist';
  END IF;
END $$;
ALTER TABLE mini_stock.reservations DROP CONSTRAINT reservations_status_check;
ALTER TABLE mini_stock.reservations ADD CONSTRAINT reservations_status_check CHECK (status = 'HELD');

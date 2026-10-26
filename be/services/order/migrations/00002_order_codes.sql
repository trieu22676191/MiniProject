-- +goose Up
CREATE SEQUENCE mini_order.order_code_seq AS bigint START WITH 1;
GRANT USAGE, SELECT ON SEQUENCE mini_order.order_code_seq TO mini_app;

-- +goose Down
DROP SEQUENCE mini_order.order_code_seq;

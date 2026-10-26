# NEO Mini

Một repository cho bài thực hành trong [neo-mini.pdf](neo-mini.pdf): Go services, Next.js web, Protobuf và hạ tầng chạy local.

```text
fe/                    # Next.js frontend
  src/app/
  src/components/
  src/lib/data/        # Hàm gọi BFF
be/                    # Go backend và hạ tầng
  services/
    order/             # Đơn hàng, duyệt đơn, tổng tiền
      cmd/server/
      internal/{domain,usecase,adapter/}
      migrations/
      queries/
    stock/             # Mặt hàng, tồn kho, giữ hàng nguyên tử
      cmd/server/
      internal/{domain,usecase,adapter/}
      migrations/
      queries/
    bff/               # HTTP/JSON cho frontend, gọi gRPC
      cmd/server/
      internal/adapter/{http,grpc}/
  pkg/{tenant,db,outbox,inbox}/
  proto/mini/{order,stock}/v1/
  gen/go/              # Mã Go sinh từ Protobuf
  deploy/              # Docker Compose và cấu hình local
```

Mỗi dịch vụ Go giữ quy tắc nghiệp vụ trong `domain` và `usecase`; `adapter` chứa giao tiếp gRPC, PostgreSQL và NATS. Một `go.mod` ở `be/` dành cho cả ba dịch vụ Go. `fe/src/lib/data` là nơi gọi BFF để các trang không gọi API trực tiếp.

## Chạy phần Ngày 1 trên Windows

Cần Go 1.27.x, Docker Desktop và PowerShell. Trong `be/`, cài đúng công cụ của đề:

```powershell
go install github.com/bufbuild/buf/cmd/buf@v1.54.0
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0
go install github.com/pressly/goose/v3/cmd/goose@v3.26.0
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
$env:PATH = (Join-Path (go env GOPATH) 'bin') + ';' + $env:PATH
```

Tạo `be/deploy/.env` từ `.env.example` và thay bốn mật khẩu mẫu. File `.env` được Git bỏ qua. Không đưa key, token, mật khẩu hoặc khóa ký thật vào code, tài liệu hay log; service đọc chúng từ biến môi trường. Từ `be/`, chạy:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/dev.ps1 up
powershell -ExecutionPolicy Bypass -File scripts/dev.ps1 migrate
powershell -ExecutionPolicy Bypass -File scripts/dev.ps1 stock
# Trong terminal khác, có thể chạy phần đọc đơn hiện có:
powershell -ExecutionPolicy Bypass -File scripts/dev.ps1 order
```

PostgreSQL mở trên `127.0.0.1:15432` (đổi bằng `POSTGRES_HOST_PORT` nếu cần); NATS trên `127.0.0.1:4222`. gRPC `order` nghe ở `127.0.0.1:50051`, `stock` ở `127.0.0.1:50052`. Migration chạy bằng `mini_owner`; service chạy bằng `mini_app` và đặt tenant trong từng giao dịch.

Lệnh kiểm phần hiện có:

```powershell
buf lint
buf generate
sqlc generate -f services/stock/sqlc.yaml
sqlc generate -f services/order/sqlc.yaml
go vet ./...
go test ./...
powershell -ExecutionPolicy Bypass -File scripts/dev.ps1 test
```

Lệnh `test` tạo database riêng, chạy migration và integration test, rồi xóa database đó. `order` hiện có `CreateOrder`, `GetOrder`, `ListOrders`, `ApproveOrder`, `RejectOrder`; `bff` và web chưa triển khai. Hướng dẫn demo đầy đủ sẽ được bổ sung cùng các phần này. Ghi chú học tập cá nhân có thể lưu trong `KNOWLEDGE.md`; file này được Git bỏ qua.

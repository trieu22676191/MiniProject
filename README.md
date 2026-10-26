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

## Chạy backend trên Windows

Cần Go 1.27.x, Docker Desktop và PowerShell. Trong `be/`, cài đúng công cụ của đề:

```powershell
go install github.com/bufbuild/buf/cmd/buf@v1.54.0
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0
go install github.com/pressly/goose/v3/cmd/goose@v3.26.0
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
$env:PATH = (Join-Path (go env GOPATH) 'bin') + ';' + $env:PATH
```

Tạo `be/deploy/.env` từ `.env.example` và thay bốn mật khẩu mẫu. File `.env` được Git bỏ qua. Không đưa key, token, mật khẩu hoặc khóa ký thật vào code, tài liệu hay log; service đọc chúng từ biến môi trường. Xem [Quy tắc bảo mật dữ liệu](QUY_TAC_BAO_MAT_DU_LIEU.md) trước khi thêm cấu hình hoặc commit.

```powershell
powershell -ExecutionPolicy Bypass -File scripts/dev.ps1 up
powershell -ExecutionPolicy Bypass -File scripts/dev.ps1 migrate
# Terminal 1: chạy stock (gRPC :50052, consumer và outbox relay)
powershell -ExecutionPolicy Bypass -File scripts/dev.ps1 stock
# Terminal 2: chạy order (gRPC :50051, consumer và outbox relay)
powershell -ExecutionPolicy Bypass -File scripts/dev.ps1 order

# Terminal 3: chạy REST BFF tại 127.0.0.1:8080
powershell -ExecutionPolicy Bypass -File scripts/dev.ps1 bff
```

PostgreSQL mở trên `127.0.0.1:15432` (đổi bằng `POSTGRES_HOST_PORT` nếu cần); NATS trên `127.0.0.1:4222`. gRPC `order` nghe ở `127.0.0.1:50051`, `stock` ở `127.0.0.1:50052`; BFF nghe ở `127.0.0.1:8080`. Migration chạy bằng `mini_owner`; service chạy bằng `mini_app` và đặt tenant trong từng giao dịch. Sau khi ba terminal backend đã chạy, mở terminal thứ tư tại `fe/` và chạy `corepack pnpm@11.0.0 install`, sau đó `corepack pnpm@11.0.0 dev`; web mở ở `http://localhost:3000`. Cần Node.js 24 theo đề.

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

Lệnh `test` tạo database riêng, chạy migration và integration test, rồi xóa database đó. `order` có `CreateOrder`, `GetOrder`, `ListOrders`, `ApproveOrder`, `RejectOrder`; duyệt đơn phát outbox, `stock` giữ hàng nguyên tử rồi trả kết quả qua JetStream để `order` cập nhật trạng thái. Cần chạy cả hai service để hoàn tất luồng sự kiện. bốn route frontend và REST BFF đã được triển khai. Ghi chú học tập cá nhân có thể lưu trong `KNOWLEDGE.md`; file này được Git bỏ qua.

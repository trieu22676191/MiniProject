# NEO Mini

Một repository cho bài thực hành trong [neo-mini.pdf](neo-mini.pdf): Go services, Next.js web, Protobuf và hạ tầng chạy local.

```text
fe/                    # Next.js frontend
  app/
  components/
  lib/data/            # Hàm gọi BFF
be/                    # Go backend và hạ tầng
  services/
    order/             # Đơn hàng, duyệt đơn, tổng tiền
      cmd/order/
      internal/{domain,usecase,adapter/}
      migrations/
      queries/
    stock/             # Mặt hàng, tồn kho, giữ hàng nguyên tử
      cmd/stock/
      internal/{domain,usecase,adapter/}
      migrations/
      queries/
    bff/               # HTTP/JSON cho frontend, gọi gRPC
      cmd/bff/
      internal/adapter/{http,grpc}/
  pkg/{tenant,db,outbox,inbox}/
  proto/mini/{order,stock}/v1/
  gen/go/              # Mã Go sinh từ Protobuf
  deploy/              # Docker Compose và cấu hình local
```

Mỗi dịch vụ Go giữ quy tắc nghiệp vụ trong `domain` và `usecase`; `adapter` chứa giao tiếp gRPC, PostgreSQL và NATS. Dự kiến dùng một `go.mod` ở `be/` cho cả ba dịch vụ Go. `fe/lib/data` là nơi gọi BFF để các trang không gọi API trực tiếp.

Các thư mục hiện có `.gitkeep` để Git theo dõi bộ khung. Cấu hình và mã nguồn sẽ được thêm khi triển khai từng phần.

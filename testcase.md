# Kịch Bản Kiểm Thử Thủ Công (Manual Test Cases) — NEO Mini

Tài liệu hướng dẫn từng bước kiểm thử giao diện và nghiệp vụ trên hệ thống NEO Mini theo chuẩn **Mục 9 (Tiêu chí đạt)** và các phần mở rộng trong [`neo-mini.pdf`](neo-mini.pdf).

---

## 1. Chuẩn Bị Trước Khi Test

1. **Khởi động hệ thống:**
   - Đảm bảo **Docker Desktop** đang chạy.
   - Nhấp đúp file [`Chay-NEO-Mini.bat`](Chay-NEO-Mini.bat).
   - Đợi 4 cửa sổ terminal xuất hiện và báo sẵn sàng:
     - `NEO Mini - Stock` (Port 50052)
     - `NEO Mini - Order` (Port 50051)
     - `NEO Mini - BFF` (Port 8080)
     - `NEO Mini - Frontend` (Port 3000)
2. **Mở trình duyệt:** Truy cập địa chỉ `http://localhost:3000`.

### Danh sách tài khoản demo và quyền hạn:
| Người dùng | Tenant | Capabilities | Mục đích test chính |
| :--- | :---: | :--- | :--- |
| **An** (`an`) | Tenant A | `order.create` | Tạo đơn hàng mới; không có quyền duyệt |
| **Bình** (`binh`) | Tenant A | `order.create`, `order.approve` | Thử nghiệm quy tắc chặn người lập tự duyệt đơn mình |
| **Chi** (`chi`) | Tenant A | `order.approve` | Duyệt hoặc từ chối các đơn do người khác lập |
| **Dũng** (`dung`) | Tenant B | `order.create`, `order.approve` | Kiểm tra cách ly dữ liệu giữa các tenant (RLS) |
| **Kho** (`kho`) | Tenant A | `stock.adjust` | Quản trị tồn kho, điều chỉnh số lượng thực tế |

---

## 2. Kịch Bản Kiểm Thử Chi Tiết (AC-01 đến AC-09 & EX)

---

### Test Case AC-01: Tạo đơn hàng 2 dòng (Giá & Tổng tiền từ Server)
- **Mục tiêu**: Xác nhận giá lấy từ kho, tính đúng tổng tiền đến từng đồng, client gửi giá cũng bị bỏ qua.
- **Các bước thực hiện**:
  1. Tại trang chủ `/`, chọn tài khoản **An**.
  2. Bấm nút **"+ Tạo đơn mới"** (hoặc vào menu **Tạo đơn**).
  3. Tại dòng 1: Chọn mặt hàng `Bàn làm việc` (SKU `BAN-01`), nhập số lượng: `2`.
  4. Tại dòng 2: Chọn mặt hàng `Ghế văn phòng` (SKU `GHE-01`), nhập số lượng: `3`.
  5. Dòng 3 để trống ("Bỏ qua dòng này").
  6. Bấm nút **"Tạo đơn"**.
- **Kết quả mong đợi**:
  - [x] Hệ thống chuyển hướng đến trang chi tiết đơn hàng `/orders/[id]`.
  - [x] Trạng thái đơn là **"Chờ duyệt"** (`PENDING_APPROVAL`).
  - [x] Mã đơn dạng `DH-00000x`.
  - [x] Đơn giá tự động lấy từ kho: `BAN-01` là `1.250.000 ₫`, `GHE-01` là `550.000 ₫`.
  - [x] Tổng tiền hiển thị chính xác: $2 \times 1.250.000 + 3 \times 550.000 = 4.150.000\text{ ₫}$.

---

### Test Case AC-02: Quyền duyệt đơn (Ẩn nút cho người không có quyền)
- **Mục tiêu**: Người không có quyền `order.approve` không thể duyệt đơn và không thấy nút Duyệt trên giao diện.
- **Các bước thực hiện**:
  1. Đang ở tài khoản **An** (chỉ có quyền `order.create`), mở chi tiết đơn hàng vừa tạo ở AC-01.
  2. Quan sát phần thao tác xử lý đơn ở cuối trang.
- **Kết quả mong đợi**:
  - [x] Không xuất hiện nút **"Duyệt đơn"** hay **"Từ chối"** cho tài khoản An.
  - [x] Danh sách `allowedActions` của đơn đối với An là rỗng `[]`.

---

### Test Case AC-03: Quy tắc cấm tự duyệt đơn (Self-Approval Denied)
- **Mục tiêu**: Người lập đơn không được tự duyệt đơn của mình, kể cả khi có quyền duyệt `order.approve`.
- **Các bước thực hiện**:
  1. Bấm **"Đổi người dùng"** $\rightarrow$ Chọn tài khoản **Bình** (có cả quyền tạo và duyệt).
  2. Bấm **"+ Tạo đơn mới"**, chọn `Bàn làm việc` số lượng `1`, bấm **"Tạo đơn"**.
  3. Sau khi tạo xong, trang chuyển đến chi tiết đơn hàng do Bình vừa tạo.
  4. Quan sát phần hành động thao tác dưới chân trang.
- **Kết quả mong đợi**:
  - [x] Bình không nhìn thấy nút **"Duyệt đơn"** hay **"Từ chối"** đối với đơn do chính Bình tạo.
  - [x] Đơn giữ nguyên trạng thái **"Chờ duyệt"** (`PENDING_APPROVAL`).

---

### Test Case AC-04: Cách ly dữ liệu giữa các công ty / Tenant (PostgreSQL RLS)
- **Mục tiêu**: Người dùng Tenant B không thể thấy hoặc truy cập đơn hàng của Tenant A.
- **Các bước thực hiện**:
  1. Copy đường link URL chi tiết đơn hàng của Tenant A vừa tạo ở các bước trên (ví dụ: `http://localhost:3000/orders/<uuid-cua-tenant-a>`).
  2. Bấm **"Đổi người dùng"** $\rightarrow$ Chọn **Dũng** (thuộc **Tenant B**).
  3. Quan sát trang danh sách đơn hàng `/orders`: Không có bất kỳ đơn hàng nào của Tenant A.
  4. Dán trực tiếp đường link URL đơn hàng của Tenant A đã copy vào thanh địa chỉ trình duyệt rồi nhấn Enter.
- **Kết quả mong đợi**:
  - [x] Trang hiển thị **404 - Không tìm thấy trang / dữ liệu**.
  - [x] Row Level Security (RLS) của PostgreSQL chặn hoàn toàn ở mức cơ sở dữ liệu.

---

### Test Case AC-05: Duyệt đơn đủ hàng $\rightarrow$ Giữ kho thành công (`RESERVED`)
- **Mục tiêu**: Khi người có quyền duyệt một đơn đủ hàng, đơn chuyển sang `APPROVED`, kho giữ hàng nguyên tử và đơn cập nhật thành `RESERVED`.
- **Các bước thực hiện**:
  1. Bấm **"Đổi người dùng"** $\rightarrow$ Chọn **Chi** (Tenant A, có quyền `order.approve`, không tạo đơn).
  2. Vào danh sách đơn hàng `/orders`, mở đơn hàng do **An** tạo ở test case AC-01.
  3. Chi nhìn thấy khối thao tác: Nút **"Duyệt đơn"** và form nhập lý do **"Từ chối"**.
  4. Bấm nút **"Duyệt đơn"**.
  5. Trang hiển thị trạng thái `Kho đang xác nhận hàng` và có nút **"Làm mới"**.
  6. Sau 1-2 giây (hoặc bấm nút "Làm mới"), quan sát trạng thái cập nhật.
- **Kết quả mong đợi**:
  - [x] Trạng thái đơn chuyển thành **"Đã giữ hàng"** (`RESERVED`) với nhãn màu xanh mint.
  - [x] Người quyết định được ghi nhận: `Người quyết định: chi`.
  - [x] Vào mục **Tồn kho** `/stock`: Số lượng tồn khả dụng của `BAN-01` và `GHE-01` giảm tương ứng với số lượng đơn đã đặt.

---

### Test Case AC-06: Duyệt đơn thiếu hàng $\rightarrow$ Hết hàng (`OUT_OF_STOCK`)
- **Mục tiêu**: Kho thiếu hàng (dù chỉ 1 dòng) thì không giữ dòng nào, đơn chuyển thành `OUT_OF_STOCK`. Theo đặc tả Section 2 & 6.1 của tài liệu, việc tạo đơn chỉ tra cứu đơn giá (`StockService.GetItems`), việc kiểm tra và giữ hàng diễn ra nguyên tử khi duyệt đơn.
- **Các bước thực hiện**:
  1. Kiểm tra tồn kho của đèn bàn `DEN-01` (dữ liệu ban đầu chỉ có 1 chiếc khả dụng).
  2. Chuyển sang tài khoản **An**, tạo đơn hàng thiếu:
     - **Cách 1 (Tạo 1 đơn đặt vượt số lượng)**: Chọn `DEN-01` với số lượng `2` $\rightarrow$ bấm **Tạo đơn** (đơn tạo thành công ở trạng thái `SUBMITTED`).
     - **Cách 2 (Tạo 2 đơn liên tiếp cạnh tranh)**: Tạo 2 đơn riêng biệt, mỗi đơn chọn `DEN-01` số lượng `1`.
  3. Chuyển sang tài khoản **Chi**:
     - *Nếu làm theo Cách 1*: Mở đơn $\rightarrow$ bấm **"Duyệt đơn"**.
     - *Nếu làm theo Cách 2*: Mở đơn thứ nhất $\rightarrow$ bấm **"Duyệt đơn"** (chuyển thành `RESERVED`). Sau đó mở đơn thứ hai $\rightarrow$ bấm **"Duyệt đơn"**.
- **Kết quả mong đợi**:
  - [x] Đơn thiếu hàng chuyển trạng thái thành **"Thiếu hàng"** (`OUT_OF_STOCK`) với nhãn cảnh báo đỏ cam.
  - [x] Hiển thị thông báo: *"Kho không đủ số lượng để giữ đơn hàng này. Không có mặt hàng nào bị giữ một phần."*
  - [x] Kiểm tra kho: Số lượng `reserved` của `DEN-01` không bao giờ vượt quá tồn thực tế.

---

### Test Case AC-07 & AC-08: Chống xử lý trùng (Inbox) & Chống bán âm khi tranh chấp song song
- **Mục tiêu**:
  - Đảm bảo hệ thống sử dụng Transactional Outbox + Inbox deduplication (`(tenant_id, consumer, event_id)`).
  - Đảm bảo khóa hàng `SELECT ... FOR UPDATE ORDER BY sku` chống race condition và deadlock.
- **Cách kiểm thử xác nhận**:
  - Chạy bộ kiểm thử tích hợp tự động với 20 cặp đơn tranh chấp đồng thời:
    ```powershell
    cd d:\MiniProject\be
    powershell -ExecutionPolicy Bypass -File scripts/dev.ps1 test
    ```
- **Kết quả mong đợi**:
  - [x] `TestReservation_AC07_DuplicateApprovedEventReservesOnce`: PASS (sự kiện lặp chỉ giữ 1 lần).
  - [x] `TestReservation_AC08_ConcurrentOrdersCannotOversell`: PASS (20 lần chạy song song, đúng 1 đơn giữ được, không bán âm).
  - [x] `TestReservation_AC08_WithoutLock_ConcurrentWritesHitConstraint`: PASS (chứng minh nếu bỏ khóa dòng thì bị lỗi tranh chấp).

---

### Test Case AC-09: Từ chối đơn & Chặn duyệt đơn đã từ chối (`REJECTED`)
- **Mục tiêu**: Người duyệt có thể từ chối đơn kèm lý do (10–500 ký tự); đơn đã từ chối thì không thể duyệt lại (`INVALID_TRANSITION`).
- **Các bước thực hiện**:
  1. Đăng nhập bằng **An**, tạo 1 đơn hàng mới bất kỳ.
  2. Đăng nhập bằng **Chi**, mở chi tiết đơn hàng đó.
  3. Tại ô "Lý do từ chối", nhập lý do hợp lệ (ví dụ: `Khách hàng yêu cầu hủy đặt hàng này`) rồi bấm **"Từ chối"**.
- **Kết quả mong đợi**:
  - [x] Trạng thái đơn chuyển thành **"Đã từ chối"** (`REJECTED`).
  - [x] Lý do từ chối được hiển thị rõ ràng trên chi tiết đơn.
  - [x] Mọi nút "Duyệt" hay "Từ chối" đều biến mất khỏi màn hình; không thể thao tác đổi trạng thái tiếp.

---

### Test Case EX-01 (Mở rộng): Hủy đơn hàng đã giữ kho & Nhả tồn kho
- **Mục tiêu**: Cho phép người tạo hoặc người duyệt hủy đơn đã `RESERVED`, phát sự kiện bù trừ để kho nhả lại hàng.
- **Các bước thực hiện**:
  1. Mở một đơn hàng đang ở trạng thái **"Đã giữ hàng"** (`RESERVED`) mà bạn có quyền (An hoặc Chi).
  2. Ở cuối trang xuất hiện nút **"Hủy đơn và nhả hàng"**.
  3. Bấm nút **"Hủy đơn và nhả hàng"**.
- **Kết quả mong đợi**:
  - [x] Trạng thái đơn chuyển sang **"Đã hủy"** (`CANCELLED`).
  - [x] Kiểm tra mục **Tồn kho** `/stock`: Số lượng tồn khả dụng được cộng lại đúng bằng số lượng hàng đã hủy.

---

### Test Case EX-03 (Mở rộng): Thủ kho điều chỉnh tồn kho
- **Mục tiêu**: Người dùng có capability `stock.adjust` được phép điều chỉnh tồn thực tế, nhưng không được phép hạ thấp hơn số đang giữ (`reserved`).
- **Các bước thực hiện**:
  1. Bấm **"Đổi người dùng"** $\rightarrow$ Chọn **Kho** (Tenant A).
  2. Hệ thống chuyển thẳng đến trang **Tồn kho** `/stock`.
  3. Chọn mặt hàng `Bàn làm việc`, nhập số lượng điều chỉnh (ví dụ: `+5` hoặc `-2`) rồi bấm **"Lưu thay đổi"**.
  4. Thử nhập một số âm lớn khiến `on_hand < reserved`.
- **Kết quả mong đợi**:
  - [x] Điều chỉnh hợp lệ: Tồn thực tế và tồn khả dụng tăng/giảm chính xác.
  - [x] Điều chỉnh vi phạm (thấp hơn hàng đang giữ): Hệ thống báo lỗi và từ chối cập nhật.

---

## 3. Bảng Tổng Hợp Kiểm Thử Nghiệm Thu

| Mã Test | Mô Tả Kịch Bản | Kết Quả Thực Tế | Đánh Giá |
| :---: | :--- | :---: | :---: |
| **AC-01** | Tạo đơn 2 dòng, tính đúng tiền từ server | Đơn tạo thành công, giá và tổng tiền chính xác | [ ] Đạt |
| **AC-02** | Ẩn nút duyệt đối với An (thiếu quyền) | Không có nút duyệt/từ chối | [ ] Đạt |
| **AC-03** | Chặn Bình tự duyệt đơn do mình lập | Báo lỗi hoặc ẩn nút, giữ nguyên PENDING_APPROVAL | [ ] Đạt |
| **AC-04** | Dũng (Tenant B) không xem được đơn Tenant A | RLS chặn, trả về 404 | [ ] Đạt |
| **AC-05** | Chi duyệt đơn đủ hàng $\rightarrow$ `RESERVED` | Chuyển RESERVED, kho giữ hàng thành công | [ ] Đạt |
| **AC-06** | Duyệt đơn thiếu hàng $\rightarrow$ `OUT_OF_STOCK` | Chuyển OUT_OF_STOCK, không giữ 1 phần | [ ] Đạt |
| **AC-07** | Nhận lặp sự kiện duyệt đơn | Inbox dedupe, chỉ giữ đúng 1 lần | [ ] Đạt |
| **AC-08** | Tranh chấp tồn 1 cái khi chạy song song | Đúng 1 đơn giữ được, không bán âm | [ ] Đạt |
| **AC-09** | Chặn duyệt đơn đã `REJECTED` | Khóa trạng thái, trả INVALID_TRANSITION | [ ] Đạt |
| **EX-01** | Hủy đơn đã giữ hàng, nhả tồn kho | Đơn CANCELLED, kho hoàn lại số lượng | [ ] Đạt |
| **EX-03** | Điều chỉnh tồn kho an toàn | Cập nhật được, không hạ thấp hơn reserved | [ ] Đạt |


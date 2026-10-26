# Quy tắc bảo mật dữ liệu và bí mật

Tài liệu này áp dụng cho toàn bộ repository NEO Mini, gồm `fe/`, `be/`, tài liệu, lịch sử Git và môi trường chạy local. Mục tiêu là ngăn API key, token, mật khẩu, khóa ký và dữ liệu nhạy cảm lọt vào mã nguồn hoặc repository.

## 1. Không đưa bí mật vào repository

- Không ghi giá trị thật của API key, access token, refresh token, mật khẩu, private key, signing key, DSN có mật khẩu hoặc cookie bí mật vào mã nguồn, file cấu hình được commit, tài liệu, ví dụ, ảnh chụp màn hình hay log.
- Không đặt bí mật trong frontend: mọi giá trị có tiền tố `NEXT_PUBLIC_` có thể được gửi tới trình duyệt. Frontend chỉ gọi API phía server; bí mật nếu có phải chỉ được đọc trong backend hoặc server runtime.
- Không dùng giá trị thật làm giá trị mặc định trong code. Nếu thiếu cấu hình bắt buộc, ứng dụng phải báo lỗi và dừng an toàn.
- Không đưa dữ liệu khách hàng, thông tin cá nhân hoặc dữ liệu production vào seed, fixture, ví dụ hoặc bản ghi demo. Chỉ dùng dữ liệu giả.

## 2. Nơi lưu và cách nạp bí mật

- Khi chạy local, lưu giá trị trong file `.env` cục bộ đã được Git ignore hoặc biến môi trường của hệ điều hành. Repository hiện ignore `.env` và `.env.*`, ngoại trừ `.env.example`.
- `.env.example` chỉ chứa tên biến và giá trị giả như `change-me`; không sao chép mật khẩu dùng thật vào file này.
- Backend đọc cấu hình từ biến môi trường. Không hard-code bí mật và không ghi lại chúng khi khởi động.
- Khi triển khai thật, dùng secret manager hoặc cơ chế quản lý bí mật của nền tảng. Cấp quyền tối thiểu, tách bí mật theo môi trường và thay khóa định kỳ theo chính sách của hệ thống.
- Không truyền bí mật qua tham số dòng lệnh nếu công cụ có thể lưu lệnh trong lịch sử shell hoặc danh sách tiến trình.

## 3. Quy tắc với Git

Trước khi stage, commit hoặc push:

1. Xem `git status` và danh sách file sẽ đưa vào commit.
2. Xem toàn bộ diff đã stage; tìm các dấu hiệu như `password`, `secret`, `token`, `api_key`, `private_key`, chuỗi kết nối và header xác thực.
3. Xác nhận `.env`, `.env.local`, file key và file cấu hình local không nằm trong danh sách được theo dõi.
4. Chỉ commit `.env.example` khi mọi giá trị trong đó là giả.

`.gitignore` chỉ ngăn Git theo dõi file mới; nó không xóa bí mật đã được commit trước đó. Nếu bí mật đã lọt vào Git, phải coi bí mật đó đã bị lộ kể cả khi xóa file trong commit sau.

## 4. Log và thông báo lỗi

- Không log request/response đầy đủ nếu có thể chứa authorization header, cookie, token, mật khẩu hoặc dữ liệu cá nhân.
- Không đưa bí mật vào message của lỗi, stack trace, telemetry, tên branch hoặc nội dung commit.
- Khi cần điều tra, log mã lỗi, loại thao tác, mã tương quan và định danh đã được phép; che hoặc loại bỏ dữ liệu nhạy cảm trước khi ghi.
- Không trả lỗi nội bộ, câu truy vấn, DSN hay thông tin kết nối cho client. Trả mã lỗi và thông báo phù hợp, giữ chi tiết kỹ thuật trong log đã làm sạch.

## 5. Dữ liệu trong ứng dụng

- Chỉ thu thập và lưu dữ liệu cần thiết cho nghiệp vụ; giới hạn quyền đọc/ghi theo tenant và vai trò.
- Không gửi bí mật hoặc dữ liệu cá nhân không cần thiết qua event, metadata gRPC, URL query, analytics hay thông báo lỗi.
- Dùng HTTPS/TLS cho kết nối qua mạng không tin cậy; không tắt xác minh chứng thư trong môi trường thật.
- Mọi API key/token phải được cấp đúng phạm vi, quyền tối thiểu và thời hạn cần thiết; không dùng chung một key giữa local, staging và production.
- Các tài khoản demo và cơ chế `X-Dev-User` của NEO Mini chỉ dành cho bài thực hành local, không phải xác thực production.

## 6. Nếu phát hiện bí mật bị lộ

1. Thu hồi hoặc xoay vòng key/token/mật khẩu ngay tại nơi đã cấp.
2. Kiểm tra log truy cập và các hệ thống có thể đã nhận bản sao của bí mật.
3. Gỡ giá trị khỏi code/config đang dùng và thay bằng cơ chế nạp bí mật an toàn.
4. Nếu bí mật đã vào Git, xử lý cả lịch sử repository và các bản sao/remote theo quy trình của dự án; xóa file ở commit mới không làm key cũ an toàn trở lại.
5. Ghi nhận sự cố và các bước khắc phục nhưng không sao chép bí mật vào báo cáo.

## 7. Checklist ngắn trước khi commit

- [ ] Không có key, token, mật khẩu, private key hoặc DSN thật trong diff.
- [ ] Không có `.env`, file local hoặc dữ liệu production trong danh sách staged.
- [ ] Mọi giá trị mẫu đều giả và được ghi rõ là mẫu.
- [ ] Không có bí mật trong log, thông báo lỗi hoặc tài liệu.
- [ ] Đã xem lại `git diff --cached` trước khi tạo commit.

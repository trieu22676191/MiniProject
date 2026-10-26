import Link from "next/link";

export default function NotFound() {
  return <div className="mx-auto max-w-lg rounded-2xl border border-line bg-white p-8 text-center"><p className="text-xs font-semibold uppercase tracking-[.18em] text-forest">404 · Không tìm thấy</p><h1 className="mt-3 text-2xl font-semibold">Đơn hàng không tồn tại</h1><p className="mt-2 text-sm text-muted">Đơn có thể không thuộc tenant hiện tại hoặc đã bị xóa.</p><Link className="mt-6 inline-flex h-10 items-center rounded-xl bg-forest px-5 text-sm font-semibold text-white hover:bg-forest-dark" href="/orders">Về danh sách đơn</Link></div>;
}

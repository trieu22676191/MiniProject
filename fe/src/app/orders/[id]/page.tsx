import Link from "next/link";
import { notFound, redirect } from "next/navigation";
import { ApiError, getOrder } from "@/lib/data/api";
import { formatMoney, statusLabel } from "@/lib/format";
import { OrderActions, RefreshWhileReserving } from "@/components/order-actions";
import { StatusPill } from "@/components/status-pill";

export default async function OrderDetailPage({ params, searchParams }: { params: Promise<{ id: string }>; searchParams: Promise<{ error?: string }> }) {
  const [{ id }, { error }] = await Promise.all([params, searchParams]);
  let order;
  try {
    order = await getOrder(id);
  } catch (cause) {
    if (cause instanceof ApiError && cause.status === 401) redirect("/");
    if (cause instanceof ApiError && cause.status === 404) notFound();
    throw cause;
  }
  return (
    <div className="mx-auto max-w-4xl">
      <RefreshWhileReserving status={order.status} />
      <Link href="/orders" className="text-sm font-medium text-muted transition hover:text-forest">← Danh sách đơn hàng</Link>
      <div className="mt-6 flex flex-col justify-between gap-4 sm:flex-row sm:items-start">
        <div><p className="text-xs font-semibold uppercase tracking-[.18em] text-forest">Chi tiết đơn hàng</p><h1 className="mt-2 text-3xl font-semibold tracking-[-.035em]">{order.code}</h1><p className="mt-2 text-sm text-muted">Người tạo: {order.createdBy}{order.decidedBy ? ` · Người quyết định: ${order.decidedBy}` : ""}</p></div>
        <StatusPill status={order.status} />
      </div>
      {error && <p className="mt-6 rounded-xl border border-[#f1d5cb] bg-[#fff7f3] px-4 py-3 text-sm text-[#a34730]">{error === "SELF_APPROVAL" ? "Bạn không thể tự duyệt đơn hàng của mình." : error === "INVALID_TRANSITION" ? "Trạng thái đơn hàng đã thay đổi, vui lòng tải lại." : error === "PERMISSION_DENIED" ? "Bạn không có quyền thực hiện thao tác này." : "Không thể xử lý đơn hàng."}</p>}
      {order.status === "APPROVED" && <div className="mt-6 flex items-start gap-3 rounded-2xl border border-[#d8e3ff] bg-[#f5f7ff] p-4 text-sm text-[#42578e]"><span className="mt-0.5 animate-pulse">●</span><span><strong className="font-semibold">Kho đang xác nhận hàng.</strong><span className="mt-1 block text-xs leading-5">Trang sẽ tự làm mới khi có kết quả giữ hàng.</span></span></div>}
      {order.status === "OUT_OF_STOCK" && <div className="mt-6 rounded-2xl border border-[#f1d5cb] bg-[#fff7f3] p-4 text-sm text-[#a34730]">Kho không đủ số lượng để giữ đơn hàng này. Không có mặt hàng nào bị giữ một phần.</div>}
      {order.status === "REJECTED" && order.rejectReason && <div className="mt-6 rounded-2xl border border-line bg-white p-4"><p className="text-xs font-semibold text-muted">Lý do từ chối</p><p className="mt-2 text-sm leading-6">{order.rejectReason}</p></div>}
      {order.status === "CANCELLED" && <div className="mt-6 rounded-2xl border border-line bg-[#f7f7f5] p-4 text-sm text-muted">Đơn hàng đã được hủy. Số lượng hàng đã giữ (nếu có) được trả lại kho.</div>}
      <div className="mt-7 grid gap-6 lg:grid-cols-[1fr_280px]">
        <section className="overflow-hidden rounded-2xl border border-line bg-white">
          <div className="border-b border-line px-5 py-4"><h2 className="font-semibold">Mặt hàng trong đơn</h2><p className="mt-1 text-xs text-muted">{order.lines.length} loại mặt hàng</p></div>
          <div className="divide-y divide-line">{order.lines.map((line) => <div className="flex items-center justify-between gap-3 px-5 py-4" key={line.sku}><div><p className="text-sm font-semibold">{line.sku}</p><p className="mt-1 text-xs text-muted">{formatMoney(line.unitPrice)} × {line.quantity}</p></div><p className="text-sm font-semibold">{formatMoney(line.lineAmount)}</p></div>)}</div>
          <div className="flex items-center justify-between bg-[#fafbf9] px-5 py-4"><span className="text-sm font-semibold">Tổng cộng</span><span className="text-lg font-bold tracking-tight text-forest">{formatMoney(order.totalAmount)}</span></div>
        </section>
        <aside className="space-y-4"><div className="rounded-2xl border border-line bg-white p-5"><p className="text-xs font-semibold uppercase tracking-[.14em] text-muted">Trạng thái</p><p className="mt-3 font-semibold">{statusLabel(order.status)}</p><p className="mt-2 text-xs leading-5 text-muted">{order.status === "PENDING_APPROVAL" ? "Đơn đang chờ người có quyền phê duyệt." : order.status === "RESERVED" ? "Tất cả mặt hàng đã được giữ cho đơn này." : order.status === "APPROVED" ? "Đơn đã duyệt và đang chờ kho xử lý." : "Thông tin đơn hàng được cập nhật theo kết quả xử lý."}</p></div></aside>
      </div>
      {order.allowedActions.length > 0 && <div className="mt-6"><OrderActions order={order} /></div>}
    </div>
  );
}

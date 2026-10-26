import Link from "next/link";
import { redirect } from "next/navigation";
import { ApiError, getMe, listOrders } from "@/lib/data/api";
import { formatMoney } from "@/lib/format";
import { StatusPill } from "@/components/status-pill";

export default async function OrdersPage() {
  let user;
  try {
    user = await getMe();
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) redirect("/");
    throw error;
  }
  if (!user.capabilities.some((capability) => capability === "order.create" || capability === "order.approve")) redirect("/stock");
  let orders;
  try {
    orders = await listOrders();
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) redirect("/");
    throw error;
  }
  const canCreateOrder = user.capabilities.includes("order.create");
  return (
    <div>
      <div className="flex flex-col justify-between gap-5 sm:flex-row sm:items-end">
        <div><p className="text-xs font-semibold uppercase tracking-[.18em] text-forest">Không gian làm việc · {user.name}</p><h1 className="mt-2 text-3xl font-semibold tracking-[-.035em]">Đơn hàng</h1><p className="mt-2 text-sm text-muted">Theo dõi và xử lý đơn hàng trong tenant của bạn.</p></div>
        <div className="flex flex-wrap gap-3">
          <Link href="/stock" className="inline-flex h-11 items-center justify-center rounded-xl border border-line bg-white px-5 text-sm font-semibold text-forest transition hover:bg-mint">Xem tồn kho</Link>
          {canCreateOrder && <Link href="/orders/new" className="inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-forest px-5 text-sm font-semibold text-white shadow-sm transition hover:bg-forest-dark"><span className="text-lg leading-none">+</span> Tạo đơn mới</Link>}
        </div>
      </div>
      <div className="mt-8 grid grid-cols-2 gap-3 sm:grid-cols-3">
        <SummaryCard label="Tổng đơn hàng" value={String(orders.length).padStart(2, "0")} />
        <SummaryCard label="Chờ xử lý" value={String(orders.filter((order) => ["PENDING_APPROVAL", "APPROVED"].includes(order.status)).length).padStart(2, "0")} />
        <SummaryCard label="Đã hoàn tất" value={String(orders.filter((order) => order.status === "RESERVED").length).padStart(2, "0")} />
      </div>
      <section className="mt-7 overflow-hidden rounded-2xl border border-line bg-white">
        <div className="flex items-center justify-between border-b border-line px-5 py-4 sm:px-6"><div><h2 className="font-semibold">Danh sách gần đây</h2><p className="mt-1 text-xs text-muted">Tối đa 50 đơn mới nhất</p></div><span className="rounded-full bg-paper px-3 py-1 text-xs font-medium text-muted">{orders.length} đơn</span></div>
        {orders.length === 0 ? <div className="px-6 py-16 text-center"><span className="mx-auto grid size-12 place-items-center rounded-2xl bg-mint text-xl text-forest">↗</span><h3 className="mt-4 font-semibold">Chưa có đơn hàng</h3><p className="mt-1 text-sm text-muted">{canCreateOrder ? "Tạo đơn đầu tiên để bắt đầu quy trình." : "Đơn hàng mới sẽ xuất hiện ở đây khi có người gửi duyệt."}</p>{canCreateOrder && <Link href="/orders/new" className="mt-5 inline-block text-sm font-semibold text-forest">Tạo đơn mới →</Link>}</div> : (
          <div className="overflow-x-auto"><table className="w-full min-w-[600px] text-left text-sm"><thead className="bg-[#fafbf9] text-xs text-muted"><tr><th className="px-6 py-3.5 font-medium">Mã đơn</th><th className="px-6 py-3.5 font-medium">Người tạo</th><th className="px-6 py-3.5 font-medium">Tổng tiền</th><th className="px-6 py-3.5 font-medium">Trạng thái</th><th className="px-6 py-3.5" /></tr></thead><tbody className="divide-y divide-line">{orders.map((order) => <tr className="transition hover:bg-[#fbfdfb]" key={order.id}><td className="px-6 py-4"><Link className="font-semibold text-forest hover:underline" href={`/orders/${order.id}`}>{order.code}</Link></td><td className="px-6 py-4 text-muted">{order.createdBy}</td><td className="px-6 py-4 font-medium">{formatMoney(order.totalAmount)}</td><td className="px-6 py-4"><StatusPill status={order.status} /></td><td className="px-6 py-4 text-right text-muted">→</td></tr>)}</tbody></table></div>
        )}
      </section>
    </div>
  );
}

function SummaryCard({ label, value }: { label: string; value: string }) {
  return <div className="rounded-2xl border border-line bg-white p-4 sm:p-5"><p className="text-xs text-muted">{label}</p><p className="mt-2 text-2xl font-semibold tracking-tight">{value}</p></div>;
}

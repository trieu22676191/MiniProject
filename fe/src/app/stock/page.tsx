import { redirect } from "next/navigation";
import { adjustStockAction } from "@/app/actions";
import { ApiError, getMe, listStockItems } from "@/lib/data/api";
import { formatMoney } from "@/lib/format";

const errorMessages: Record<string, string> = {
  INVALID_ARGUMENT: "Nhập mã hàng và số lượng nguyên khác 0.",
  PERMISSION_DENIED: "Tài khoản hiện tại không có quyền điều chỉnh tồn kho.",
  FAILED_PRECONDITION: "Số lượng tồn mới không thể thấp hơn số hàng đang được giữ.",
  UNAVAILABLE: "Chưa kết nối được dịch vụ kho. Hãy thử lại sau.",
};

export default async function StockPage({ searchParams }: { searchParams: Promise<{ error?: string; updated?: string }> }) {
  const [{ error, updated }, user, items] = await Promise.all([
    searchParams,
    getMe().catch((cause: unknown) => {
      if (cause instanceof ApiError && cause.status === 401) redirect("/");
      throw cause;
    }),
    listStockItems().catch((cause: unknown) => {
      if (cause instanceof ApiError && cause.status === 401) redirect("/");
      throw cause;
    }),
  ]);
  const canAdjust = user.capabilities.includes("stock.adjust");

  return (
    <div className="mx-auto max-w-5xl">
      <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-end">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[.18em] text-forest">Kho hàng</p>
          <h1 className="mt-2 text-3xl font-semibold tracking-[-.035em]">Tồn kho hiện tại</h1>
          <p className="mt-2 text-sm text-muted">Theo dõi số lượng có sẵn và hàng đã giữ cho các đơn.</p>
        </div>
        <span className="rounded-full border border-line bg-white px-3 py-2 text-xs font-medium text-muted">{items.length} mặt hàng</span>
      </div>

      {error && <p role="alert" className="mt-6 rounded-xl border border-[#f1d5cb] bg-[#fff7f3] px-4 py-3 text-sm text-[#a34730]">{errorMessages[error] ?? "Không thể cập nhật tồn kho. Hãy tải lại và thử lại."}</p>}
      {updated && <p role="status" className="mt-6 rounded-xl border border-[#cce6d5] bg-mint px-4 py-3 text-sm text-forest">Đã cập nhật tồn kho cho {updated}.</p>}

      <section className="mt-6 overflow-hidden rounded-2xl border border-line bg-white">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[680px] text-left">
            <thead className="bg-[#fafbf9] text-[11px] font-semibold uppercase tracking-wider text-muted">
              <tr><th className="px-5 py-4">Mặt hàng</th><th className="px-4 py-4">Đơn giá</th><th className="px-4 py-4">Tồn kho</th><th className="px-4 py-4">Đang giữ</th><th className="px-4 py-4">Có thể dùng</th>{canAdjust && <th className="px-5 py-4">Điều chỉnh</th>}</tr>
            </thead>
            <tbody className="divide-y divide-line">
              {items.map((item) => (
                <tr key={item.sku}>
                  <td className="px-5 py-4"><p className="text-sm font-semibold">{item.name}</p><p className="mt-1 text-xs text-muted">{item.sku}</p></td>
                  <td className="px-4 py-4 text-sm">{formatMoney(item.unitPrice)}</td>
                  <td className="px-4 py-4 text-sm font-medium">{item.onHand}</td>
                  <td className="px-4 py-4 text-sm text-muted">{item.reserved}</td>
                  <td className="px-4 py-4 text-sm font-semibold text-forest">{item.available}</td>
                  {canAdjust && <td className="px-5 py-4"><form action={adjustStockAction} className="flex items-center gap-2"><input type="hidden" name="sku" value={item.sku} /><label className="sr-only" htmlFor={`delta-${item.sku}`}>Số lượng thay đổi cho {item.name}</label><input id={`delta-${item.sku}`} name="deltaOnHand" type="number" step="1" required placeholder="± số lượng" className="h-10 w-28 rounded-lg border border-line px-3 text-sm outline-none focus:border-forest" /><button type="submit" className="h-10 rounded-lg bg-forest px-3 text-xs font-semibold text-white transition hover:bg-forest-dark">Cập nhật</button></form></td>}
                </tr>
              ))}
              {items.length === 0 && <tr><td colSpan={canAdjust ? 6 : 5} className="px-5 py-12 text-center text-sm text-muted">Chưa có mặt hàng trong kho.</td></tr>}
            </tbody>
          </table>
        </div>
      </section>

      <p className="mt-4 text-xs leading-5 text-muted">{canAdjust ? "Nhập số nguyên dương để tăng hoặc số âm để giảm. Hệ thống không cho giảm thấp hơn lượng hàng đang được giữ." : "Bạn đang ở chế độ xem. Chọn tài khoản Kho để điều chỉnh tồn."}</p>
    </div>
  );
}

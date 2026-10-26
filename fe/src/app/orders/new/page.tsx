import Link from "next/link";
import { redirect } from "next/navigation";
import { createOrderAction } from "@/app/actions";
import { ApiError, getMe, listStockItems } from "@/lib/data/api";
import { formatMoney } from "@/lib/format";

export default async function NewOrderPage({ searchParams }: { searchParams: Promise<{ error?: string }> }) {
  const [{ error }] = await Promise.all([searchParams]);
  let items;
  try {
    items = await listStockItems();
    await getMe();
  } catch (cause) {
    if (cause instanceof ApiError && cause.status === 401) redirect("/");
    throw cause;
  }
  const hasStock = items.some((item) => item.available > 0);
  return (
    <div className="mx-auto max-w-3xl">
      <Link href="/orders" className="text-sm font-medium text-muted transition hover:text-forest">← Quay lại đơn hàng</Link>
      <div className="mt-6"><p className="text-xs font-semibold uppercase tracking-[.18em] text-forest">Đơn hàng mới</p><h1 className="mt-2 text-3xl font-semibold tracking-[-.035em]">Tạo đơn hàng</h1><p className="mt-2 text-sm text-muted">Chọn mặt hàng và số lượng. Giá sẽ được lấy trực tiếp từ kho.</p></div>
      {error && <p className="mt-6 rounded-xl border border-[#f1d5cb] bg-[#fff7f3] px-4 py-3 text-sm text-[#a34730]">{error === "NOT_FOUND" ? "Một hoặc nhiều mặt hàng không còn tồn tại." : error === "PERMISSION_DENIED" ? "Tài khoản hiện tại không có quyền tạo đơn." : "Không thể tạo đơn hàng. Hãy kiểm tra thông tin và thử lại."}</p>}
      {!hasStock ? <div className="mt-8 rounded-2xl border border-line bg-white p-8 text-center"><h2 className="font-semibold">Kho hiện chưa có hàng khả dụng</h2><p className="mt-2 text-sm text-muted">Bạn có thể quay lại sau khi tồn kho được cập nhật.</p></div> : (
        <form action={createOrderAction} className="mt-8 rounded-2xl border border-line bg-white p-5 sm:p-7">
          <div className="mb-5 flex items-center justify-between"><h2 className="font-semibold">Mặt hàng cần đặt</h2><span className="text-xs text-muted">Tối đa 3 dòng</span></div>
          <div className="space-y-3">{[0, 1, 2].map((row) => (
            <div className="grid gap-3 rounded-xl border border-line bg-[#fcfdfb] p-3 sm:grid-cols-[1fr_110px] sm:items-center" key={row}>
              <label className="block"><span className="mb-1.5 block text-[11px] font-medium text-muted">Mặt hàng {row + 1}</span><select name="sku" defaultValue={row === 0 ? "" : ""} required={row === 0} className="h-11 w-full rounded-lg border border-line bg-white px-3 text-sm outline-none focus:border-forest"><option value="">{row === 0 ? "Chọn một mặt hàng" : "Bỏ qua dòng này"}</option>{items.filter((item) => item.available > 0).map((item) => <option value={item.sku} key={item.sku}>{item.name} · {item.available} còn</option>)}</select></label>
              <label className="block"><span className="mb-1.5 block text-[11px] font-medium text-muted">Số lượng</span><input name="quantity" type="number" min="1" step="1" defaultValue={row === 0 ? 1 : ""} className="h-11 w-full rounded-lg border border-line bg-white px-3 text-sm outline-none focus:border-forest" /></label>
            </div>
          ))}</div>
          <div className="mt-5 rounded-xl bg-mint/60 p-4"><p className="text-xs font-semibold text-forest">Giá tham khảo từ kho</p><div className="mt-2 flex flex-wrap gap-x-5 gap-y-2">{items.filter((item) => item.available > 0).map((item) => <span className="text-xs text-[#496455]" key={item.sku}>{item.sku} · {formatMoney(item.unitPrice)}</span>)}</div></div>
          <div className="mt-7 flex flex-col-reverse justify-end gap-3 border-t border-line pt-5 sm:flex-row"><Link href="/orders" className="inline-flex h-11 items-center justify-center rounded-xl border border-line px-5 text-sm font-semibold text-muted hover:bg-paper">Hủy</Link><button type="submit" className="h-11 rounded-xl bg-forest px-6 text-sm font-semibold text-white transition hover:bg-forest-dark">Gửi duyệt đơn hàng →</button></div>
        </form>
      )}
    </div>
  );
}

import Link from "next/link";
import { redirect } from "next/navigation";
import { ApiError, getMe, listStockItems } from "@/lib/data/api";
import { NewOrderForm } from "@/components/new-order-form";

export default async function NewOrderPage({ searchParams }: { searchParams: Promise<{ error?: string }> }) {
  const [{ error }] = await Promise.all([searchParams]);
  let items;
  let user;
  try {
    user = await getMe();
  } catch (cause) {
    if (cause instanceof ApiError && cause.status === 401) redirect("/");
    throw cause;
  }
  if (!user.capabilities.includes("order.create")) redirect(user.capabilities.includes("order.approve") ? "/orders" : "/stock");
  try {
    items = await listStockItems();
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
        <NewOrderForm items={items} />
      )}
    </div>
  );
}

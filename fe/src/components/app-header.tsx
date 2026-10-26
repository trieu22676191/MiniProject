import Link from "next/link";
import { cookies } from "next/headers";
import { getMe } from "@/lib/data/api";

export async function AppHeader() {
  const userCookie = (await cookies()).get("dev_user")?.value;
  let name = "Chưa đăng nhập";
  let canViewOrders = false;
  let canCreateOrder = false;
  if (userCookie) {
    try {
      const user = await getMe();
      name = user.name;
      canViewOrders = user.capabilities.some((capability) => capability === "order.create" || capability === "order.approve");
      canCreateOrder = user.capabilities.includes("order.create");
    } catch {
      name = "Phiên không hợp lệ";
    }
  }

  return (
    <header className="border-b border-line bg-white">
      <div className="mx-auto flex h-[76px] max-w-6xl items-center justify-between px-5 sm:px-8">
        <Link href={canViewOrders ? "/orders" : "/stock"} className="flex items-center gap-3" aria-label="NEO Mini, về trang chính">
          <span className="grid size-10 place-items-center rounded-xl bg-forest text-sm font-black tracking-tight text-white">N.</span>
          <span>
            <span className="block text-sm font-bold tracking-tight">NEO Mini</span>
            <span className="block text-[11px] text-muted">Đơn hàng & tồn kho</span>
          </span>
        </Link>
        <nav className="hidden items-center gap-7 text-sm font-medium text-muted sm:flex">
          {canViewOrders && <Link className="transition hover:text-forest" href="/orders">Đơn hàng</Link>}
          {canCreateOrder && <Link className="transition hover:text-forest" href="/orders/new">Tạo đơn</Link>}
          <Link className="transition hover:text-forest" href="/stock">Tồn kho</Link>
          <Link className="transition hover:text-forest" href="/">Đổi người dùng</Link>
        </nav>
        <div className="flex items-center gap-3 rounded-full border border-line bg-paper py-1.5 pl-1.5 pr-3">
          <span className="grid size-8 place-items-center rounded-full bg-mint text-xs font-bold text-forest">{name.slice(0, 1)}</span>
          <span className="max-w-24 truncate text-xs font-semibold">{name}</span>
        </div>
      </div>
      <nav className="mx-auto flex max-w-6xl items-center gap-6 overflow-x-auto border-t border-line px-5 py-3 text-sm font-medium text-muted sm:hidden" aria-label="Điều hướng chính">
        {canViewOrders && <Link className="shrink-0 hover:text-forest" href="/orders">Đơn hàng</Link>}
        {canCreateOrder && <Link className="shrink-0 hover:text-forest" href="/orders/new">Tạo đơn</Link>}
        <Link className="shrink-0 hover:text-forest" href="/stock">Tồn kho</Link>
        <Link className="shrink-0 hover:text-forest" href="/">Đổi người dùng</Link>
      </nav>
    </header>
  );
}

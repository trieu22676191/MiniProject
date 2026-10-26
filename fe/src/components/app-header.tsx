import Link from "next/link";
import { cookies } from "next/headers";
import { getMe } from "@/lib/data/api";

export async function AppHeader() {
  const userCookie = (await cookies()).get("dev_user")?.value;
  let name = "Chưa đăng nhập";
  let canAdjustStock = false;
  if (userCookie) {
    try {
      const user = await getMe();
      name = user.name;
      canAdjustStock = user.capabilities.includes("stock.adjust");
    } catch {
      name = "Phiên không hợp lệ";
    }
  }

  return (
    <header className="border-b border-line bg-white">
      <div className="mx-auto flex h-[76px] max-w-6xl items-center justify-between px-5 sm:px-8">
        <Link href="/orders" className="flex items-center gap-3" aria-label="NEO Mini, về đơn hàng">
          <span className="grid size-10 place-items-center rounded-xl bg-forest text-sm font-black tracking-tight text-white">N.</span>
          <span>
            <span className="block text-sm font-bold tracking-tight">NEO Mini</span>
            <span className="block text-[11px] text-muted">Đơn hàng & tồn kho</span>
          </span>
        </Link>
        <nav className="hidden items-center gap-7 text-sm font-medium text-muted sm:flex">
          <Link className="transition hover:text-forest" href="/orders">Đơn hàng</Link>
          <Link className="transition hover:text-forest" href="/orders/new">Tạo đơn</Link>
          {canAdjustStock && <Link className="transition hover:text-forest" href="/stock">Tồn kho</Link>}
          <Link className="transition hover:text-forest" href="/">Đổi người dùng</Link>
        </nav>
        <div className="flex items-center gap-3 rounded-full border border-line bg-paper py-1.5 pl-1.5 pr-3">
          <span className="grid size-8 place-items-center rounded-full bg-mint text-xs font-bold text-forest">{name.slice(0, 1)}</span>
          <span className="max-w-24 truncate text-xs font-semibold">{name}</span>
        </div>
      </div>
    </header>
  );
}

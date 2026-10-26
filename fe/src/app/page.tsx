import { selectUserAction } from "./actions";

const users = [
  { id: "an", name: "An", role: "Tạo đơn hàng", tone: "A" },
  { id: "binh", name: "Bình", role: "Tạo và duyệt đơn", tone: "B" },
  { id: "chi", name: "Chi", role: "Duyệt đơn hàng", tone: "C" },
  { id: "dung", name: "Dũng", role: "Tenant B · Tạo và duyệt", tone: "D" },
  { id: "kho", name: "Kho", role: "Tenant A · Điều chỉnh tồn kho", tone: "K" },
];

export default async function Home({ searchParams }: { searchParams: Promise<{ error?: string }> }) {
  const { error } = await searchParams;
  return (
    <div className="grid min-h-[70vh] items-center gap-12 lg:grid-cols-[0.9fr_1.1fr]">
      <section className="max-w-lg">
        <p className="mb-5 inline-flex items-center gap-2 rounded-full border border-[#cce6d5] bg-mint px-3 py-1.5 text-xs font-semibold text-forest">
          <span className="size-1.5 rounded-full bg-forest" /> Môi trường thực hành
        </p>
        <h1 className="text-4xl font-semibold leading-[1.12] tracking-[-0.04em] sm:text-5xl">Chào mừng đến với <span className="text-forest">NEO Mini.</span></h1>
        <p className="mt-5 max-w-md text-base leading-7 text-muted">Chọn một tài khoản demo để bắt đầu. Mỗi người dùng có quyền và phạm vi tenant riêng.</p>
        <div className="mt-8 flex items-center gap-3 text-xs text-muted"><span className="h-px w-9 bg-[#c5d3c9]" /> Chọn tài khoản để tiếp tục</div>
      </section>
      <section className="rounded-[28px] border border-line bg-white p-5 shadow-[0_24px_70px_-40px_rgba(21,52,37,.28)] sm:p-8">
        <div className="mb-6 flex items-start justify-between">
          <div><p className="text-xs font-semibold uppercase tracking-[.18em] text-forest">Tài khoản demo</p><h2 className="mt-2 text-xl font-semibold tracking-tight">Bạn là ai hôm nay?</h2></div>
          <span className="rounded-lg bg-paper px-2.5 py-1.5 text-[11px] font-semibold text-muted">5 người dùng</span>
        </div>
        {error && <p className="mb-4 rounded-xl bg-red-50 px-4 py-3 text-sm text-red-700">Không nhận diện được người dùng. Hãy chọn lại.</p>}
        <div className="space-y-3">
          {users.map((user) => (
            <form action={selectUserAction} key={user.id}>
              <input type="hidden" name="user" value={user.id} />
              <button className="group flex w-full items-center gap-4 rounded-2xl border border-line p-4 text-left transition hover:border-[#acd4bb] hover:bg-[#f8fcf8] focus-visible:outline-2 focus-visible:outline-forest" type="submit">
                <span className={`grid size-12 shrink-0 place-items-center rounded-2xl text-sm font-bold ${user.id === "dung" ? "bg-[#f8ead6] text-[#9b611e]" : "bg-mint text-forest"}`}>{user.tone}</span>
                <span className="min-w-0 flex-1"><span className="block font-semibold">{user.name}</span><span className="mt-1 block text-xs text-muted">{user.role}</span></span>
                <span className="text-xl text-[#a8b4ac] transition group-hover:translate-x-1 group-hover:text-forest">→</span>
              </button>
            </form>
          ))}
        </div>
        <p className="mt-5 text-[11px] leading-5 text-muted">Tài khoản chỉ dùng cho bài thực hành, không phải đăng nhập thật.</p>
      </section>
    </div>
  );
}

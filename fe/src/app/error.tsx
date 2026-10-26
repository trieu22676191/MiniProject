"use client";

export default function ErrorPage({ reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return <div className="mx-auto max-w-xl rounded-2xl border border-line bg-white p-8 text-center"><span className="mx-auto grid size-12 place-items-center rounded-full bg-[#fff0eb] text-xl text-[#b24d32]">!</span><h1 className="mt-4 text-xl font-semibold">Không tải được dữ liệu</h1><p className="mt-2 text-sm leading-6 text-muted">Dịch vụ có thể đang tạm ngắt. Hãy thử tải lại sau ít phút.</p><button className="mt-6 h-10 rounded-xl bg-forest px-5 text-sm font-semibold text-white hover:bg-forest-dark" onClick={reset}>Thử lại</button></div>;
}

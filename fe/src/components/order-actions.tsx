"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { approveOrderAction, rejectOrderAction } from "@/app/actions";
import type { Order } from "@/lib/data/types";

export function OrderActions({ order }: { order: Order }) {
  const canApprove = order.allowedActions.includes("APPROVE");
  const canReject = order.allowedActions.includes("REJECT");
  if (!canApprove && !canReject) return null;

  return (
    <section className="rounded-2xl border border-line bg-white p-5 sm:p-6">
      <p className="text-sm font-semibold">Quyết định đơn hàng</p>
      <p className="mt-1 text-xs leading-5 text-muted">Người tạo đơn không thể tự duyệt hoặc từ chối đơn của mình.</p>
      <div className="mt-5 flex flex-col gap-3 sm:flex-row">
        {canApprove && <form action={approveOrderAction}><input type="hidden" name="id" value={order.id} /><button className="h-11 w-full rounded-xl bg-forest px-5 text-sm font-semibold text-white hover:bg-forest-dark sm:w-auto" type="submit">Duyệt đơn</button></form>}
        {canReject && <form action={rejectOrderAction} className="flex flex-1 flex-col gap-2 sm:flex-row"><input type="hidden" name="id" value={order.id} /><input name="reason" minLength={10} maxLength={500} required placeholder="Lý do từ chối (10–500 ký tự)" className="h-11 min-w-0 flex-1 rounded-xl border border-line px-3 text-sm outline-none focus:border-forest" /><button className="h-11 rounded-xl border border-line px-5 text-sm font-semibold text-[#9b4937] hover:bg-[#fff7f3]" type="submit">Từ chối</button></form>}
      </div>
    </section>
  );
}

export function RefreshWhileReserving({ status }: { status: string }) {
  const router = useRouter();
  useEffect(() => {
    if (status !== "APPROVED") return;
    const timer = window.setInterval(() => router.refresh(), 1800);
    const stop = window.setTimeout(() => window.clearInterval(timer), 30_000);
    return () => { window.clearInterval(timer); window.clearTimeout(stop); };
  }, [router, status]);
  return null;
}

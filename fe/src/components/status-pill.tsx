import { statusLabel } from "@/lib/format";

export function StatusPill({ status }: { status: string }) {
  const styles: Record<string, string> = {
    PENDING_APPROVAL: "bg-[#fff3dd] text-[#9a5c0a]",
    APPROVED: "bg-[#eaf0ff] text-[#455eae]",
    RESERVED: "bg-mint text-forest",
    OUT_OF_STOCK: "bg-[#fff0eb] text-[#b24d32]",
    REJECTED: "bg-[#f2eeee] text-[#725e5e]",
    CANCELLED: "bg-[#f2eeee] text-[#725e5e]",
  };
  return <span className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-semibold ${styles[status] ?? "bg-paper text-muted"}`}><span className="size-1.5 rounded-full bg-current" />{statusLabel(status)}</span>;
}

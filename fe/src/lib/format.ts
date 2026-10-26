import type { Money } from "./data/types";

export function formatMoney(money: Money) {
  const amount = Number(money.amount);
  if (!Number.isFinite(amount)) return `${money.amount} ${money.currencyCode}`;
  return new Intl.NumberFormat("vi-VN", {
    style: "currency",
    currency: money.currencyCode,
    maximumFractionDigits: 0,
  }).format(amount);
}

export function statusLabel(status: string) {
  const labels: Record<string, string> = {
    PENDING_APPROVAL: "Chờ duyệt",
    APPROVED: "Chờ kho xác nhận",
    RESERVED: "Đã giữ hàng",
    OUT_OF_STOCK: "Thiếu hàng",
    REJECTED: "Đã từ chối",
  };
  return labels[status] ?? status;
}

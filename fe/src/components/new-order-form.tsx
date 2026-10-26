"use client";

import { useState } from "react";
import Link from "next/link";
import { createOrderAction } from "@/app/actions";
import type { StockItem } from "@/lib/data/types";
import { formatMoney } from "@/lib/format";

export function NewOrderForm({ items }: { items: StockItem[] }) {
  const [selectedSkus, setSelectedSkus] = useState(["", "", ""]);
  const [quantities, setQuantities] = useState(["1", "", ""]);
  const [cappedRows, setCappedRows] = useState([false, false, false]);
  const availableItems = items.filter((item) => item.available > 0);

  function selectItem(row: number, sku: string) {
    setSelectedSkus((current) => current.map((value, index) => index === row ? sku : value));
    setQuantities((current) => current.map((value, index) => index === row ? (sku ? "1" : "") : value));
    setCappedRows((current) => current.map((value, index) => index === row ? false : value));
  }

  function changeQuantity(row: number, rawValue: string, available: number) {
    if (rawValue === "") {
      setQuantities((current) => current.map((value, index) => index === row ? "" : value));
      setCappedRows((current) => current.map((value, index) => index === row ? false : value));
      return;
    }
    if (!/^\d+$/.test(rawValue) || !Number.isSafeInteger(Number(rawValue))) return;
    const requested = Number(rawValue);
    setQuantities((current) => current.map((value, index) => index === row ? rawValue : value));
    setCappedRows((current) => current.map((value, index) => index === row ? requested > available : value));
  }

  return (
    <form action={createOrderAction} className="mt-8 rounded-2xl border border-line bg-white p-5 sm:p-7">
      <div className="mb-5 flex items-center justify-between"><h2 className="font-semibold">Mặt hàng cần đặt</h2><span className="text-xs text-muted">Tối đa 3 dòng</span></div>
      <div className="space-y-3">{selectedSkus.map((sku, row) => {
        const selectedItem = availableItems.find((item) => item.sku === sku);
        return (
          <div className="grid gap-3 rounded-xl border border-line bg-[#fcfdfb] p-3 sm:grid-cols-[1fr_140px] sm:items-center" key={row}>
            <label className="block">
              <span className="mb-1.5 block text-[11px] font-medium text-muted">Mặt hàng {row + 1}</span>
              <select name="sku" value={sku} required={row === 0} onChange={(event) => selectItem(row, event.target.value)} className="h-11 w-full rounded-lg border border-line bg-white px-3 text-sm outline-none focus:border-forest">
                <option value="">{row === 0 ? "Chọn một mặt hàng" : "Bỏ qua dòng này"}</option>
                {availableItems.map((item) => {
                  const alreadySelected = selectedSkus.some((s, i) => i !== row && s === item.sku);
                  return (
                    <option value={item.sku} key={item.sku} disabled={alreadySelected}>
                      {item.name} · {item.available} còn{alreadySelected ? " (đã chọn)" : ""}
                    </option>
                  );
                })}
              </select>
            </label>
            <label className="block">
              <span className="mb-1.5 block text-[11px] font-medium text-muted">Số lượng{selectedItem ? ` · tồn khả dụng ${selectedItem.available}` : ""}</span>
              <input name="quantity" type="number" min="1" step="1" required={Boolean(sku)} value={quantities[row]} onChange={(event) => selectedItem && changeQuantity(row, event.target.value, selectedItem.available)} readOnly={!sku} className="h-11 w-full rounded-lg border border-line bg-white px-3 text-sm outline-none focus:border-forest read-only:bg-paper read-only:text-muted" />
              {cappedRows[row] && <span className="mt-1 block text-xs text-[#9b4937]" role="status">Số lượng vượt quá tồn khả dụng ({selectedItem?.available}).</span>}
            </label>
          </div>
        );
      })}</div>
      <div className="mt-5 rounded-xl bg-mint/60 p-4"><p className="text-xs font-semibold text-forest">Giá tham khảo từ kho</p><div className="mt-2 flex flex-wrap gap-x-5 gap-y-2">{availableItems.map((item) => <span className="text-xs text-[#496455]" key={item.sku}>{item.sku} · {formatMoney(item.unitPrice)}</span>)}</div></div>
      <div className="mt-7 flex flex-col-reverse justify-end gap-3 border-t border-line pt-5 sm:flex-row"><Link href="/orders" className="inline-flex h-11 items-center justify-center rounded-xl border border-line px-5 text-sm font-semibold text-muted hover:bg-paper">Hủy</Link><button type="submit" className="h-11 rounded-xl bg-forest px-6 text-sm font-semibold text-white transition hover:bg-forest-dark">Tạo đơn</button></div>
    </form>
  );
}

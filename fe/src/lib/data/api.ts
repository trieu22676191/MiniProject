import { cookies } from "next/headers";
import type { CurrentUser, Order, StockItem } from "./types";

const bffURL = process.env.BFF_URL ?? "http://127.0.0.1:8080";

export class ApiError extends Error {
  constructor(
    message: string,
    readonly code: string,
    readonly status: number,
  ) {
    super(message);
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const cookieStore = await cookies();
  const user = cookieStore.get("dev_user")?.value;
  if (!user) throw new ApiError("Vui lòng chọn người dùng.", "UNAUTHENTICATED", 401);

  const response = await fetch(`${bffURL}${path}`, {
    ...init,
    cache: "no-store",
    headers: {
      "Content-Type": "application/json",
      "X-Dev-User": user,
      ...init?.headers,
    },
  });
  const body = (await response.json()) as T | { code: string; message: string };
  if (!response.ok) {
    const error = body as { code: string; message: string };
    throw new ApiError(error.message, error.code, response.status);
  }
  return body as T;
}

export async function getMe() {
  return request<CurrentUser>("/api/me");
}

export async function listOrders() {
  const result = await request<{ orders: Order[] }>("/api/orders");
  return result.orders;
}

export async function getOrder(id: string) {
  const result = await request<{ order: Order }>(`/api/orders/${encodeURIComponent(id)}`);
  return result.order;
}

export async function listStockItems() {
  const result = await request<{ items: StockItem[] }>("/api/stock-items");
  return result.items;
}

export async function submit<T>(path: string, method: "POST", body?: unknown) {
  return request<T>(path, { method, body: body === undefined ? undefined : JSON.stringify(body) });
}

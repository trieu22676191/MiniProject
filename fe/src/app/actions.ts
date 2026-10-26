"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { ApiError, submit } from "@/lib/data/api";

const knownUsers = new Set(["an", "binh", "chi", "dung", "kho"]);

export async function selectUserAction(formData: FormData) {
  const user = String(formData.get("user") ?? "");
  if (!knownUsers.has(user)) redirect("/?error=unknown-user");
  const cookieStore = await cookies();
  cookieStore.set("dev_user", user, {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 60 * 60 * 24 * 7,
  });
  redirect("/orders");
}

function actionError(error: unknown, fallback: string): never {
  if (error instanceof ApiError) redirect(`${fallback}?error=${encodeURIComponent(error.code)}`);
  redirect(`${fallback}?error=UNAVAILABLE`);
}

export async function createOrderAction(formData: FormData) {
  const skus = formData.getAll("sku").map(String);
  const quantities = formData.getAll("quantity").map((value) => Number(value));
  const lines = skus
    .map((sku, index) => ({ sku, quantity: quantities[index] }))
    .filter((line) => line.sku && Number.isInteger(line.quantity) && line.quantity > 0);
  let response: { order: { id: string } };
  try {
    response = await submit<{ order: { id: string } }>("/api/orders", "POST", { lines });
  } catch (error) {
    actionError(error, "/orders/new");
  }
  redirect(`/orders/${response.order.id}`);
}

export async function approveOrderAction(formData: FormData) {
  const id = String(formData.get("id") ?? "");
  try {
    await submit(`/api/orders/${encodeURIComponent(id)}/approve`, "POST");
  } catch (error) {
    actionError(error, `/orders/${id}`);
  }
  redirect(`/orders/${id}`);
}

export async function rejectOrderAction(formData: FormData) {
  const id = String(formData.get("id") ?? "");
  const reason = String(formData.get("reason") ?? "");
  try {
    await submit(`/api/orders/${encodeURIComponent(id)}/reject`, "POST", { reason });
  } catch (error) {
    actionError(error, `/orders/${id}`);
  }
  redirect(`/orders/${id}`);
}

export async function cancelOrderAction(formData: FormData) {
  const id = String(formData.get("id") ?? "");
  try {
    await submit(`/api/orders/${encodeURIComponent(id)}/cancel`, "POST");
  } catch (error) {
    actionError(error, `/orders/${id}`);
  }
  redirect(`/orders/${id}`);
}

export async function adjustStockAction(formData: FormData) {
  const sku = String(formData.get("sku") ?? "");
  const deltaOnHand = Number(formData.get("deltaOnHand"));
  if (!sku || !Number.isSafeInteger(deltaOnHand) || deltaOnHand === 0) {
    redirect("/stock?error=INVALID_ARGUMENT");
  }
  try {
    await submit(`/api/stock-items/${encodeURIComponent(sku)}/adjust`, "POST", { deltaOnHand });
  } catch (error) {
    actionError(error, "/stock");
  }
  redirect(`/stock?updated=${encodeURIComponent(sku)}`);
}

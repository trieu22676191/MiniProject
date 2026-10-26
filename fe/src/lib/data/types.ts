export type Money = { amount: string; currencyCode: string };

export type OrderLine = {
  sku: string;
  quantity: number;
  unitPrice: Money;
  lineAmount: Money;
};

export type Order = {
  id: string;
  code: string;
  status: string;
  createdBy: string;
  decidedBy: string;
  rejectReason: string;
  totalAmount: Money;
  lines: OrderLine[];
  allowedActions: string[];
};

export type StockItem = {
  sku: string;
  name: string;
  unitPrice: Money;
  onHand: number;
  reserved: number;
  available: number;
};

export type CurrentUser = {
  id: string;
  name: string;
  tenantId: string;
  capabilities: string[];
};

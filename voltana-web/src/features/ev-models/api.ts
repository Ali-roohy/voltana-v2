import { api } from "@/lib/api";

export interface EVModel {
  id: string;
  name_fa: string;
  name_en: string;
  brand: string | null;
  battery_capacity_kwh: number | null;
  range_km: number | null;
  chemistry: string | null;
  created_at: string;
}

interface ListResponse<T> {
  items: T[];
  limit: number;
  offset: number;
  total: number;
}

// The catalog is small (~12 seeded); fetch up to 100 and filter client-side.
export async function listEVModels(): Promise<EVModel[]> {
  const res = await api.get<ListResponse<EVModel>>("/v1/ev-models?limit=100");
  return res.items;
}

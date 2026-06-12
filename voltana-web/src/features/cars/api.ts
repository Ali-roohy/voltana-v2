import { api } from "@/lib/api";

// spec_overrides (TASK-0034) is the user's diff from the linked catalog car —
// echoed verbatim by the API; merge with the catalog entry client-side.
// PUT /v1/cars/:id is FULL-REPLACE: always send catalog_car_id+spec_overrides
// back on edit or they get wiped.
export type SpecOverrides = Record<string, string | number>;

export interface Car {
  id: string;
  ev_model_id: string | null;
  catalog_car_id: string | null;
  spec_overrides: SpecOverrides;
  name: string;
  license_plate: string | null;
  odometer_km: number;
  created_at: string;
  updated_at: string;
}

export interface CarInput {
  name: string;
  ev_model_id?: string | null;
  catalog_car_id?: string | null;
  spec_overrides?: SpecOverrides;
  license_plate?: string | null;
  odometer_km?: number;
}

interface ListResponse<T> {
  items: T[];
  limit: number;
  offset: number;
  total: number;
}

export async function listCars(): Promise<Car[]> {
  const res = await api.get<ListResponse<Car>>("/v1/cars?limit=100");
  return res.items;
}

export const createCar = (input: CarInput) => api.post<Car>("/v1/cars", input);
export const updateCar = (id: string, input: CarInput) => api.put<Car>(`/v1/cars/${id}`, input);
export const deleteCar = (id: string) => api.del<void>(`/v1/cars/${id}`);

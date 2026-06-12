import { api } from "@/lib/api";

// One entry of the rich EV catalog from GET /v1/cars/catalog (TASK-0033).
// Mirrors domain.CatalogCar — nullable specs come back as null, never 0.
export interface CatalogCar {
  id: string;
  name_fa: string;
  name_en: string;
  brand: string | null;
  body_style_fa: string | null;
  class: string | null;
  body_type: string | null;
  segment: string | null;
  country: string | null;
  importer: string | null;
  platform: string | null;
  battery_capacity_kwh: number | null;
  battery_voltage: string | null;
  usable_kwh: number | null;
  cell_brand: string | null;
  cell_type: string | null;
  cooling: string | null;
  range_km: number | null;
  range_standard: string | null;
  consumption_kwh_per_100km: number | null;
  motor_power_kw: number | null;
  torque_nm: number | null;
  motor_count: number | null;
  motor_type: string | null;
  acceleration_0_100_s: number | null;
  max_speed_kmh: number | null;
  drivetrain: string | null;
  ac_charge_kw: number | null;
  ac_connector: string | null;
  dc_charge_kw: number | null;
  dc_connector: string | null;
  fast_charge_window: string | null;
  fast_charge_to_80_min: number | null;
  v2l: string | null;
  v2g: string | null;
  ota: string | null;
  adas_level: string | null;
  radar_count: number | null;
  camera_count: number | null;
  weight_kg: number | null;
  trunk_liters: number | null;
  notes: string | null;
  exterior_colors: string[];
  interior_colors: string[];
  img_url: string | null;
  created_at: string;
  updated_at: string;
}

export async function getCatalog(): Promise<CatalogCar[]> {
  const res = await api.get<{ cars: CatalogCar[] }>("/v1/cars/catalog");
  return res.cars;
}

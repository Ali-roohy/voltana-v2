import type { CatalogCar } from "./api";

// Editable catalog spec fields for the customize modal (TASK-0034), organized
// in the same 6 sections as the detail drawer. Keys mirror the server-side
// override whitelist in car_service.go — keep the two in sync. Colors are
// handled by dedicated pickers (exterior_color / interior_color), not here.
export interface SpecField {
  key: string;
  label: string;
  kind: "number" | "string";
}

export interface SpecFieldSection {
  title: string;
  fields: SpecField[];
}

export const SPEC_FIELD_SECTIONS: SpecFieldSection[] = [
  {
    title: "عمومی",
    fields: [
      { key: "name_fa", label: "نام فارسی", kind: "string" },
      { key: "name_en", label: "نام انگلیسی", kind: "string" },
      { key: "brand", label: "برند", kind: "string" },
      { key: "body_style_fa", label: "بدنه", kind: "string" },
      { key: "class", label: "کلاس", kind: "string" },
      { key: "body_type", label: "نوع بدنه", kind: "string" },
      { key: "segment", label: "سطح بازار", kind: "string" },
      { key: "country", label: "کشور", kind: "string" },
      { key: "importer", label: "واردکننده", kind: "string" },
      { key: "platform", label: "پلتفرم", kind: "string" },
    ],
  },
  {
    title: "باتری",
    fields: [
      { key: "battery_capacity_kwh", label: "ظرفیت (kWh)", kind: "number" },
      { key: "usable_kwh", label: "قابل استفاده (kWh)", kind: "number" },
      { key: "battery_voltage", label: "ولتاژ", kind: "string" },
      { key: "cell_brand", label: "برند سلول", kind: "string" },
      { key: "cell_type", label: "نوع سلول", kind: "string" },
      { key: "cooling", label: "خنک‌کاری", kind: "string" },
      { key: "range_km", label: "برد (km)", kind: "number" },
      { key: "range_standard", label: "استاندارد برد", kind: "string" },
      { key: "consumption_kwh_per_100km", label: "مصرف (kWh/100km)", kind: "number" },
    ],
  },
  {
    title: "موتور و عملکرد",
    fields: [
      { key: "motor_power_kw", label: "توان (kW)", kind: "number" },
      { key: "torque_nm", label: "گشتاور (Nm)", kind: "number" },
      { key: "motor_count", label: "تعداد موتور", kind: "number" },
      { key: "motor_type", label: "نوع موتور", kind: "string" },
      { key: "acceleration_0_100_s", label: "شتاب ۰→۱۰۰ (ثانیه)", kind: "number" },
      { key: "max_speed_kmh", label: "حداکثر سرعت (km/h)", kind: "number" },
      { key: "drivetrain", label: "دیفرانسیل", kind: "string" },
    ],
  },
  {
    title: "شارژ",
    fields: [
      { key: "ac_charge_kw", label: "توان AC (kW)", kind: "number" },
      { key: "ac_connector", label: "کانکتور AC", kind: "string" },
      { key: "dc_charge_kw", label: "توان DC (kW)", kind: "number" },
      { key: "dc_connector", label: "کانکتور DC", kind: "string" },
      { key: "fast_charge_window", label: "بازه شارژ سریع", kind: "string" },
      { key: "fast_charge_to_80_min", label: "زمان شارژ سریع (دقیقه)", kind: "number" },
      { key: "v2l", label: "V2L", kind: "string" },
      { key: "v2g", label: "V2G", kind: "string" },
      { key: "ota", label: "OTA", kind: "string" },
    ],
  },
  {
    title: "ADAS",
    fields: [
      { key: "adas_level", label: "سطح ADAS", kind: "string" },
      { key: "radar_count", label: "تعداد رادار", kind: "number" },
      { key: "camera_count", label: "تعداد دوربین", kind: "number" },
    ],
  },
  {
    title: "راحتی و ابعاد",
    fields: [
      { key: "weight_kg", label: "وزن (kg)", kind: "number" },
      { key: "trunk_liters", label: "صندوق (لیتر)", kind: "number" },
      { key: "notes", label: "توضیحات", kind: "string" },
    ],
  },
];

/** The catalog's value for a spec field, as the string an <input> shows. */
export function catalogValueOf(car: CatalogCar, key: string): string {
  const v = (car as unknown as Record<string, unknown>)[key];
  if (v === null || v === undefined) return "";
  return String(v);
}

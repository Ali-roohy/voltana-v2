#!/usr/bin/env python3
"""TASK-0033 — convert .ai/data/car_verified__4_.xlsx into migrations/000016_ev_catalog.up.sql.

Stdlib only (no pandas/openpyxl — the workbook stores cells as inline strings, so
zipfile + ElementTree is enough and nothing has to be pip-installed on the host).

Source layout (verified 2026-06-11):
  - Sheet1, range A2:AQ25 — row 2 is the header, rows 3..25 are the 23 cars.
  - 43 columns; headers contain embedded newlines/legend text, so columns are
    mapped BY INDEX (0-based) below — do not match on header strings.
  - Numeric cells may carry units in the text ("266 Nm", "7.5s", "452 L").
  - Sheet2 ("Verification Notes") is a correction log — not read.

Usage:  python3 scripts/seed-ev-catalog.py   (from the repo root)
"""

import re
import sys
import zipfile
from xml.etree import ElementTree as ET

XLSX = ".ai/data/car_verified__4_.xlsx"
OUT = "migrations/000016_ev_catalog.up.sql"
NS = "{http://schemas.openxmlformats.org/spreadsheetml/2006/main}"

# 0-based column index → (sql column, kind). Kinds: text, num, int, fa_list ("،"
# separated), slash_list ("/" separated). Order here is the INSERT column order.
COLUMNS = [
    (0, "name_fa", "text"),
    (1, "name_en", "text"),
    (2, "brand", "text"),
    (3, "body_style_fa", "text"),          # بدنه (e.g. شاسی‌بلند)
    (4, "class", "text"),                  # کلاس (A..F)
    (5, "body_type", "text"),              # نوع بدنه (SUV/Sedan/…)
    (6, "segment", "text"),                # سطح بازار
    (7, "country", "text"),
    (8, "importer", "text"),
    (9, "platform", "text"),
    (10, "battery_capacity_kwh", "num"),
    (11, "battery_voltage", "text"),       # "355V", "400V/800V" — kept as text
    (12, "usable_kwh", "num"),
    (13, "cell_brand", "text"),
    (14, "cell_type", "text"),             # LFP/NMC/NCA
    (15, "cooling", "text"),
    (16, "range_km", "int"),
    (17, "range_standard", "text"),        # CLTC/WLTP/EPA/NEDC
    (18, "consumption_kwh_per_100km", "num"),
    (19, "motor_power_kw", "num"),
    (20, "torque_nm", "int"),              # "266 Nm"
    (21, "motor_count", "int"),
    (22, "motor_type", "text"),            # PMSM…
    (23, "acceleration_0_100_s", "num"),   # "7.5s"
    (24, "max_speed_kmh", "int"),
    (25, "drivetrain", "text"),            # FWD/RWD/AWD
    (26, "ac_charge_kw", "num"),           # 6.6 / 11
    (27, "ac_connector", "text"),
    (28, "dc_charge_kw", "num"),
    (29, "dc_connector", "text"),
    (30, "fast_charge_window", "text"),    # "30-80%"
    (31, "fast_charge_min", "int"),
    (32, "v2l", "text"),                   # بله/خیر/نامشخص — kept as text
    (33, "v2g", "text"),
    (34, "ota", "text"),
    (35, "adas_level", "text"),            # "L2"
    (36, "radar_count", "int"),
    (37, "camera_count", "int"),
    (38, "weight_kg", "int"),
    (39, "trunk_liters", "int"),           # "452 L"
    (40, "notes", "text"),
    (41, "exterior_colors", "fa_list"),    # "سفید، مشکی، خاکستری"
    (42, "interior_colors", "slash_list"), # "مشکی / مشکی-خاکستری"
]

UNKNOWN = {"", "-", "—", "نامشخص", "ناموجود", "N/A", "n/a"}
NUM_RE = re.compile(r"[-+]?\d+(?:\.\d+)?")


def cells(row):
    out = []
    for c in row.findall(f"{NS}c"):
        if c.get("t") == "inlineStr":
            out.append("".join(t.text or "" for t in c.iter(f"{NS}t")))
        else:
            v = c.find(f"{NS}v")
            out.append(v.text if v is not None else "")
    return out


def sql_text(raw):
    s = " ".join(raw.split())  # collapse newlines/double spaces
    if s in UNKNOWN:
        return "NULL"
    return "'" + s.replace("'", "''") + "'"


def sql_number(raw, integer=False):
    if raw.strip() in UNKNOWN:
        return "NULL"
    m = NUM_RE.search(raw)
    if not m:
        return "NULL"
    val = m.group(0)
    if integer:
        return str(round(float(val)))
    return val


def sql_array(raw, sep_chars):
    s = " ".join(raw.split())
    if s in UNKNOWN:
        return "ARRAY[]::text[]"
    parts = [p.strip() for p in re.split("[" + sep_chars + "]", s) if p.strip()]
    if not parts:
        return "ARRAY[]::text[]"
    return "ARRAY[" + ", ".join("'" + p.replace("'", "''") + "'" for p in parts) + "]"


def slug(name_en):
    s = re.sub(r"[^a-z0-9]+", "-", name_en.lower()).strip("-")
    return s or "car"


def main():
    z = zipfile.ZipFile(XLSX)
    root = ET.fromstring(z.read("xl/worksheets/sheet1.xml"))
    rows = root.findall(f".//{NS}row")
    data = [cells(r) for r in rows[1:]]  # rows[0] is the header
    if len(data) != 23:
        sys.exit(f"expected 23 data rows, got {len(data)}")

    col_names = [c[1] for c in COLUMNS] + ["img_url"]
    inserts = []
    for row in data:
        if len(row) != 43:
            sys.exit(f"expected 43 cells, got {len(row)} in row {row[:2]}")
        vals = []
        for idx, _, kind in COLUMNS:
            raw = row[idx]
            if kind == "text":
                vals.append(sql_text(raw))
            elif kind == "num":
                vals.append(sql_number(raw))
            elif kind == "int":
                vals.append(sql_number(raw, integer=True))
            elif kind == "fa_list":
                vals.append(sql_array(raw, "،,"))
            elif kind == "slash_list":
                vals.append(sql_array(raw, "/،"))
        vals.append("'/catalog/cars/" + slug(row[1]) + ".jpg'")
        inserts.append("    (" + ", ".join(vals) + ")")

    header = """-- TASK-0033 — EV catalog reference data (read-only via the API).
-- Like ev_models this table has no owner: every authed user can read it and it
-- is only written by migrations. GENERATED FILE — regenerate with
--   python3 scripts/seed-ev-catalog.py
-- after changing .ai/data/car_verified__4_.xlsx; do not hand-edit the seed rows.

CREATE TABLE ev_catalog (
    id                        UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name_fa                   VARCHAR(255) NOT NULL,
    name_en                   VARCHAR(255) NOT NULL UNIQUE,
    brand                     VARCHAR(100),
    body_style_fa             VARCHAR(100),
    class                     VARCHAR(20),
    body_type                 VARCHAR(50),
    segment                   VARCHAR(100),
    country                   VARCHAR(100),
    importer                  VARCHAR(255),
    platform                  VARCHAR(100),
    battery_capacity_kwh      NUMERIC(10,2) CHECK (battery_capacity_kwh IS NULL OR battery_capacity_kwh > 0),
    battery_voltage           VARCHAR(50),
    usable_kwh                NUMERIC(10,2) CHECK (usable_kwh IS NULL OR usable_kwh > 0),
    cell_brand                VARCHAR(100),
    cell_type                 VARCHAR(50),
    cooling                   VARCHAR(100),
    range_km                  INT CHECK (range_km IS NULL OR range_km > 0),
    range_standard            VARCHAR(20),
    consumption_kwh_per_100km NUMERIC(10,2),
    motor_power_kw            NUMERIC(8,2),
    torque_nm                 INT,
    motor_count               INT,
    motor_type                VARCHAR(100),
    acceleration_0_100_s      NUMERIC(5,2),
    max_speed_kmh             INT,
    drivetrain                VARCHAR(20),
    ac_charge_kw              NUMERIC(6,2),
    ac_connector              VARCHAR(50),
    dc_charge_kw              NUMERIC(6,2),
    dc_connector              VARCHAR(50),
    fast_charge_window        VARCHAR(50),
    fast_charge_min           INT,
    v2l                       VARCHAR(20),
    v2g                       VARCHAR(20),
    ota                       VARCHAR(20),
    adas_level                VARCHAR(20),
    radar_count               INT,
    camera_count              INT,
    weight_kg                 INT,
    trunk_liters              INT,
    notes                     TEXT,
    exterior_colors           TEXT[] NOT NULL DEFAULT '{}',
    interior_colors           TEXT[] NOT NULL DEFAULT '{}',
    img_url                   VARCHAR(255),
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ev_catalog_brand     ON ev_catalog (brand);
CREATE INDEX idx_ev_catalog_body_type ON ev_catalog (body_type);

CREATE TRIGGER ev_catalog_updated_at
    BEFORE UPDATE ON ev_catalog
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO ev_catalog (""" + ", ".join(col_names) + """) VALUES
"""

    with open(OUT, "w", encoding="utf-8") as f:
        f.write(header)
        f.write(",\n".join(inserts))
        f.write(";\n")
    print(f"wrote {OUT}: 23 rows, {len(col_names)} columns each")


if __name__ == "__main__":
    main()

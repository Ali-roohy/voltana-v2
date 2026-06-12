-- TASK-0034 — link user cars to the EV catalog with per-car spec overrides.
-- spec_overrides stores ONLY the diff from the linked catalog car (JSONB), e.g.
-- {"exterior_color":"آبی متالیک","battery_capacity_kwh":60}. Effective specs are
-- merged client-side; the API echoes the column verbatim.

ALTER TABLE cars
    ADD COLUMN catalog_car_id UUID REFERENCES ev_catalog(id) ON DELETE SET NULL,
    ADD COLUMN spec_overrides JSONB NOT NULL DEFAULT '{}'::jsonb;

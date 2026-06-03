-- Charging stations (TASK-0013) — shared reference data (NOT user-owned), so no
-- user_id; readable by every authed user. Unlike the read-only ev_models catalog,
-- stations are mutable (admin CRUD) so they carry updated_at + the shared
-- set_updated_at() trigger. The lat/lng CHECKs back the service-level bounds
-- validation (defense in depth, mirroring the SOC CHECKs on charging_sessions).
-- No status/availability/price columns — real-time availability is out of scope.

CREATE TABLE charging_stations (
    id              UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255)     NOT NULL,
    latitude        DOUBLE PRECISION NOT NULL CHECK (latitude  >= -90  AND latitude  <= 90),
    longitude       DOUBLE PRECISION NOT NULL CHECK (longitude >= -180 AND longitude <= 180),
    address         VARCHAR(500),
    connector_types VARCHAR(255),   -- CSV e.g. "CCS2,Type2"
    power_kw        INT              CHECK (power_kw IS NULL OR power_kw > 0),
    operator        VARCHAR(255),
    created_at      TIMESTAMPTZ      NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ      NOT NULL DEFAULT now()
);

CREATE TRIGGER charging_stations_updated_at
    BEFORE UPDATE ON charging_stations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

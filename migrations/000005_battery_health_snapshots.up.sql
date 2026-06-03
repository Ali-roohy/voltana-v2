-- Battery health snapshots — one row per recompute (a history series, so TASK-0008
-- can chart the SOH trend). SOH is estimated from charging history via the delta-SOC
-- method in internal/service/analytics_service.go. user_id is denormalized from the
-- car so the repository can enforce ownership without a join.

CREATE TABLE battery_health_snapshots (
    id                     UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    car_id                 UUID         NOT NULL REFERENCES cars(id)  ON DELETE CASCADE,
    user_id                UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    soh_pct                DECIMAL(5,2) NOT NULL CHECK (soh_pct > 0 AND soh_pct <= 100),
    estimated_capacity_kwh DECIMAL(6,2) NOT NULL,
    nominal_capacity_kwh   DECIMAL(6,2) NOT NULL,
    sample_session_count   INT          NOT NULL,
    confidence             VARCHAR(10)  NOT NULL CHECK (confidence IN ('low', 'medium', 'high')),
    method                 VARCHAR(20)  NOT NULL DEFAULT 'delta_soc',
    computed_at            TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX bhs_car_computed_idx ON battery_health_snapshots(car_id, computed_at DESC);

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Auto-update updated_at on any mutable table
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- -------------------------------------------------------
-- Users
-- -------------------------------------------------------
CREATE TABLE users (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email             VARCHAR(255) UNIQUE NOT NULL,
    password_hash     VARCHAR(255) NOT NULL,
    is_email_verified BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TRIGGER users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- -------------------------------------------------------
-- EV Models  (reference data — no user_id)
-- -------------------------------------------------------
CREATE TABLE ev_models (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name_fa              VARCHAR(255) NOT NULL,
    name_en              VARCHAR(255) NOT NULL,
    brand                VARCHAR(255),
    battery_capacity_kwh DECIMAL(6,2),
    range_km             INT,
    chemistry            VARCHAR(10)  CHECK (chemistry IN ('LFP', 'NMC', 'NCA')),
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX ev_models_name_fa_fts ON ev_models USING gin(to_tsvector('simple', name_fa));
CREATE INDEX ev_models_name_en_fts ON ev_models USING gin(to_tsvector('simple', name_en));

-- -------------------------------------------------------
-- Cars
-- -------------------------------------------------------
CREATE TABLE cars (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ev_model_id   UUID         REFERENCES ev_models(id) ON DELETE SET NULL,
    name          VARCHAR(255) NOT NULL,
    license_plate VARCHAR(50),
    odometer_km   INT          NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX cars_user_id_idx ON cars(user_id);

CREATE TRIGGER cars_updated_at
    BEFORE UPDATE ON cars
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- -------------------------------------------------------
-- Charging Sessions
-- -------------------------------------------------------
CREATE TABLE charging_sessions (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    car_id      UUID         NOT NULL REFERENCES cars(id) ON DELETE CASCADE,
    started_at  TIMESTAMPTZ  NOT NULL,
    ended_at    TIMESTAMPTZ,
    location    VARCHAR(255),
    kwh_charged DECIMAL(8,3),
    start_soc   INT          CHECK (start_soc  >= 0 AND start_soc  <= 100),
    end_soc     INT          CHECK (end_soc    >= 0 AND end_soc    <= 100),
    cost        DECIMAL(10,2),
    notes       TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX charging_sessions_user_id_idx   ON charging_sessions(user_id);
CREATE INDEX charging_sessions_car_id_idx    ON charging_sessions(car_id);
CREATE INDEX charging_sessions_started_at_idx ON charging_sessions(started_at DESC);

CREATE TRIGGER charging_sessions_updated_at
    BEFORE UPDATE ON charging_sessions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- -------------------------------------------------------
-- User Settings  (one row per user, auto-created on first GET)
-- -------------------------------------------------------
CREATE TABLE user_settings (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID         UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    default_car_id UUID         REFERENCES cars(id) ON DELETE SET NULL,
    peak_rate      DECIMAL(10,4) NOT NULL DEFAULT 0,
    mid_rate       DECIMAL(10,4) NOT NULL DEFAULT 0,
    offpeak_rate   DECIMAL(10,4) NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TRIGGER user_settings_updated_at
    BEFORE UPDATE ON user_settings
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

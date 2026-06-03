-- Admin authorization (TASK-0013). A boolean flag on users gates the station
-- write endpoints (POST/PUT/DELETE /v1/stations) via the AdminOnly middleware,
-- which checks this column fresh on every write (not baked into the JWT) so a
-- revoked admin loses write access immediately. Defaults false: the first admin
-- is promoted out-of-band with `UPDATE users SET is_admin = true WHERE email = …`.

ALTER TABLE users ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT false;

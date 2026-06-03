-- Remove only the seeded demo rows by their exact names (leaves admin-created
-- stations untouched).
DELETE FROM charging_stations WHERE name IN (
    'ایستگاه شارژ ولنجک',
    'ایستگاه شارژ نیاوران',
    'ایستگاه شارژ سعادت آباد',
    'ایستگاه شارژ میدان آزادی',
    'ایستگاه شارژ اکباتان'
);

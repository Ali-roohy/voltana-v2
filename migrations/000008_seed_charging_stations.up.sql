-- Demo charging stations (TASK-0013) so the map renders markers immediately,
-- before any admin user exists (admin POST is chicken-and-egg until an operator
-- promotes the first admin out-of-band). Coordinates reuse the Tehran sample set
-- previously hardcoded in the frontend pages/Map.tsx; the dropped status/price
-- fields are not columns (real-time availability is out of scope).

INSERT INTO charging_stations (name, latitude, longitude, address, connector_types, power_kw, operator) VALUES
    ('ایستگاه شارژ ولنجک',      35.8063, 51.4036, 'تهران، ولنجک، خیابان شهید بهشتی', 'CCS2',       50,  'Voltana'),
    ('ایستگاه شارژ نیاوران',     35.8131, 51.4697, 'تهران، نیاوران، میدان نیاوران',   'Type2',      22,  'Voltana'),
    ('ایستگاه شارژ سعادت آباد',  35.7619, 51.3753, 'تهران، سعادت آباد، بلوار دریا',   'CCS2,CHAdeMO', 120, 'Voltana'),
    ('ایستگاه شارژ میدان آزادی', 35.6997, 51.3380, 'تهران، میدان آزادی',             'Type2',      11,  'Voltana'),
    ('ایستگاه شارژ اکباتان',     35.7006, 51.3089, 'تهران، اکباتان، فاز یک',         'CCS2',       50,  'Voltana');

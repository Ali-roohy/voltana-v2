-- Seed the shared EV catalog. The cars/ev_models tables already exist (000001);
-- this migration only adds reference data.
--
-- Idempotency: add a UNIQUE key on name_en so re-running the seed is safe via
-- ON CONFLICT DO NOTHING. (If the full Supabase export turns out to have duplicate
-- English names, switch to a dedicated slug column — see TASK-0003.)
ALTER TABLE ev_models ADD CONSTRAINT ev_models_name_en_key UNIQUE (name_en);

-- Minimal starter set (~12 common models). Full Supabase import tracked as a
-- data/docs follow-up per TASK-0003.
INSERT INTO ev_models (name_fa, name_en, brand, battery_capacity_kwh, range_km, chemistry) VALUES
('تسلا مدل ۳',        'Tesla Model 3',     'Tesla',       60.00, 491, 'LFP'),
('تسلا مدل وای',       'Tesla Model Y',     'Tesla',       75.00, 533, 'NMC'),
('بی‌وای‌دی آتو ۳',     'BYD Atto 3',        'BYD',         60.48, 420, 'LFP'),
('بی‌وای‌دی سیل',       'BYD Seal',          'BYD',         82.50, 570, 'LFP'),
('نیسان لیف',          'Nissan Leaf',       'Nissan',      40.00, 270, 'NMC'),
('هیوندای آیونیک ۵',   'Hyundai Ioniq 5',   'Hyundai',     72.60, 481, 'NMC'),
('کیا ای‌وی۶',         'Kia EV6',           'Kia',         77.40, 528, 'NMC'),
('فولکس‌واگن آی‌دی۴',   'Volkswagen ID.4',   'Volkswagen',  77.00, 520, 'NMC'),
('ام‌جی ۴',            'MG 4',              'MG',          64.00, 450, 'LFP'),
('شورولت بولت',        'Chevrolet Bolt',    'Chevrolet',   65.00, 416, 'NMC'),
('رنو زوئی',           'Renault Zoe',       'Renault',     52.00, 395, 'NMC'),
('پولستار ۲',          'Polestar 2',        'Polestar',    78.00, 540, 'NMC')
ON CONFLICT (name_en) DO NOTHING;

-- TASK-0033 — EV catalog reference data (read-only via the API).
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

INSERT INTO ev_catalog (name_fa, name_en, brand, body_style_fa, class, body_type, segment, country, importer, platform, battery_capacity_kwh, battery_voltage, usable_kwh, cell_brand, cell_type, cooling, range_km, range_standard, consumption_kwh_per_100km, motor_power_kw, torque_nm, motor_count, motor_type, acceleration_0_100_s, max_speed_kmh, drivetrain, ac_charge_kw, ac_connector, dc_charge_kw, dc_connector, fast_charge_window, fast_charge_min, v2l, v2g, ota, adas_level, radar_count, camera_count, weight_kg, trunk_liters, notes, exterior_colors, interior_colors, img_url) VALUES
    ('تویوتا بی‌زد4ایکس', 'Toyota bZ4X FWD', 'Toyota', 'شاسی‌بلند', 'D', 'SUV', 'میان‌رده رو به لوکس', 'چین (FAW Toyota)', 'واردکنندگان متفرقه (سامانه یکپارچه)', 'e-TNGA', 66.7, '355V', 64, 'CATL', 'NMC', 'مایع‌خنک', 615, 'CLTC', 13.1, 150, 266, 1, 'PMSM', 7.5, 160, 'FWD', 6.6, 'GB/T', 90, 'GB/T', '30-80%', 30, 'خیر', 'خیر', 'بله', 'L2', 1, 1, 1930, 452, 'نسخه وارداتی رایج', ARRAY['سفید', 'مشکی', 'خاکستری', 'نقره‌ای'], ARRAY['مشکی', 'مشکی-خاکستری'], '/catalog/cars/toyota-bz4x-fwd.jpg'),
    ('فولکس واگن آی‌دی4', 'Volkswagen ID.4 Crozz Lite', 'Volkswagen', 'شاسی‌بلند', 'C', 'SUV', 'میان‌رده', 'چین', 'ماموت خودرو', 'MEB', 55.7, '400V', 52, 'CATL', 'NMC', 'مایع‌خنک', 425, 'CLTC', 13.8, 125, 310, 1, 'PMSM', 8.5, 160, 'RWD', 7, 'GB/T', 100, 'GB/T', '10-80%', 40, 'خیر', 'خیر', 'بله', 'L2', 3, 4, 1990, 543, 'نسخه ماموت خودرو', ARRAY['سفید', 'خاکستری', 'آبی نفتی (بخشنامه ماموت/راین)'], ARRAY['مشکی', 'مشکی-سفید دو رنگ'], '/catalog/cars/volkswagen-id-4-crozz-lite.jpg'),
    ('فولکس واگن آی‌دی4', 'Volkswagen ID.4 Crozz Pro', 'Volkswagen', 'شاسی‌بلند', 'C', 'SUV', 'میان‌رده', 'چین', 'ماموت خودرو', 'MEB', 84.8, '400V', 77, 'CATL', 'NMC', 'مایع‌خنک', 600, 'CLTC', 14.5, 150, 310, 1, 'PMSM', 8.5, 160, 'RWD', 7, 'GB/T', 120, 'GB/T', '10-80%', 40, 'خیر', 'خیر', 'بله', 'L2', 3, 4, 2120, 543, 'نسخه فول آپشن', ARRAY['سفید', 'خاکستری', 'آبی نفتی (بخشنامه ماموت/راین)'], ARRAY['مشکی', 'مشکی-سفید دو رنگ'], '/catalog/cars/volkswagen-id-4-crozz-pro.jpg'),
    ('هوندا ENS1', 'Honda e:NS1', 'Honda', 'شاسی‌بلند', 'B', 'SUV', 'میان‌رده', 'چین', 'معین خودرو', 'e:N Architecture F', 68.8, '353V', 65, 'CATL', 'NMC', 'مایع‌خنک', 510, 'CLTC', 13.8, 150, 310, 1, 'PMSM', 7.6, 150, 'FWD', 6.6, 'GB/T', 78, 'GB/T', '30-80%', 40, 'خیر', 'خیر', 'بله', 'L2', 1, 1, 1730, 361, 'معین خودرو', ARRAY['سفید', 'خاکستری', 'مشکی', 'آبی'], ARRAY['مشکی', 'مشکی-قهوه‌ای'], '/catalog/cars/honda-e-ns1.jpg'),
    ('اسکای‌ول ET5', 'Skywell ET5', 'Skywell', 'شاسی‌بلند', 'D', 'SUV', 'لوکس', 'چین', 'نبکا (Nebka)', 'CE Platform', 72, '400V', 70, NULL, 'NMC', 'مایع‌خنک', 520, 'NEDC', 15.2, 150, 320, 1, 'PMSM', 9.6, 150, 'FWD', 6.6, 'GB/T', 80, 'GB/T', '30-80%', 38, 'خیر', 'خیر', 'بله', 'L2', 3, 4, 1830, 467, 'نبکا', ARRAY['سفید', 'خاکستری', 'آبی', 'سبز تیره'], ARRAY['مشکی', 'قهوه‌ای-کرم'], '/catalog/cars/skywell-et5.jpg'),
    ('آئودی Q5 e-tron', 'Audi Q5 e-tron 40', 'Audi', 'شاسی‌بلند', 'D', 'SUV', 'لوکس', 'چین', 'واردکننده رسمی (سامانه یکپارچه)', 'MEB (Volkswagen)', 84.8, '400V', 79.7, 'CATL', 'NMC', 'Heat Pump', 560, 'CLTC', 16, 150, 310, 1, 'PMSM', 9.3, 160, 'RWD', 11, 'GB/T', 100, 'GB/T', '10-80%', 35, 'خیر', 'خیر', 'بله', 'L2+', 5, 4, 2325, 550, 'واردات جدید', ARRAY['سفید', 'مشکی', 'خاکستری', 'آبی'], ARRAY['مشکی', 'مشکی-قهوه‌ای'], '/catalog/cars/audi-q5-e-tron-40.jpg'),
    ('آئودی Q5 e-tron', 'Audi Q5 e-tron 50 Quattro', 'Audi', 'شاسی‌بلند', 'D', 'SUV', 'لوکس', 'چین', 'واردکننده رسمی (سامانه یکپارچه)', 'MEB (Volkswagen)', 84.8, '400V', 79.7, 'CATL', 'NMC', 'Heat Pump', 520, 'CLTC', 17.5, 225, 460, 2, 'PMSM', 6.7, 160, 'AWD', 11, 'GB/T', 100, 'GB/T', '10-80%', 35, 'خیر', 'خیر', 'بله', 'L2+', 5, 4, 2410, 550, 'AWD', ARRAY['سفید', 'مشکی', 'خاکستری', 'آبی'], ARRAY['مشکی', 'مشکی-قهوه‌ای'], '/catalog/cars/audi-q5-e-tron-50-quattro.jpg'),
    ('جی‌ای‌سی آیون Y', 'GAC Aion Y Plus 63', 'GAC', 'کراس‌اوور', 'C', 'Crossover', 'میان‌رده', 'چین', 'واردکننده متفرقه (سامانه یکپارچه)', 'GEP 2.0', 63.2, '400V', 60, 'CALB / Aion (Magazine)', 'LFP', 'مایع‌خنک', 490, 'CLTC', 13.5, 150, 225, 1, 'PMSM', 8.5, 150, 'FWD', 6.6, 'GB/T', 75, 'GB/T', '30-80%', 55, 'بله', 'خیر', 'بله', 'L2', 3, 4, 1735, 361, 'نسخه رایج', ARRAY['سفید', 'خاکستری', 'آبی روشن', 'سبز'], ARRAY['مشکی', 'خاکستری دو رنگ'], '/catalog/cars/gac-aion-y-plus-63.jpg'),
    ('جی‌ای‌سی آیون Y', 'GAC Aion Y Plus 69', 'GAC', 'کراس‌اوور', 'C', 'Crossover', 'میان‌رده', 'چین', 'واردکننده متفرقه (سامانه یکپارچه)', 'GEP 2.0', 69.90000000000001, '400V', 66, 'CALB / Aion (Magazine)', 'LFP', 'مایع‌خنک', 610, 'CLTC', 13, 150, 225, 1, 'PMSM', 8.5, 150, 'FWD', 6.6, 'GB/T', 75, 'GB/T', '30-80%', 55, 'بله', 'خیر', 'بله', 'L2', 5, 6, 1735, 361, 'یکی از اقتصادی‌ترین کراس‌اوورهای برقی وارداتی ایران', ARRAY['سفید', 'خاکستری', 'آبی روشن', 'سبز'], ARRAY['مشکی', 'خاکستری دو رنگ'], '/catalog/cars/gac-aion-y-plus-69.jpg'),
    ('بی‌وای‌دی آتو3', 'BYD Atto 3', 'BYD', 'کراس‌اوور', 'C', 'Crossover', 'میان‌رده', 'چین', 'واردکننده متفرقه (نسخه صادراتی)', 'e-Platform 3.0', 60.48, '400V', 60.48, 'Byd Blade', 'LFP Blade', 'Heat Pump+مایع‌خنک', 420, 'WLTP', 15.6, 150, 310, 1, 'PMSM', 7.3, 160, 'FWD', 11, 'Type 2', 110, 'CCS2', '30-80%', 29, 'بله', 'خیر', 'بله', 'L2', 5, 6, 1750, 440, 'واردات محدود', ARRAY['سفید', 'خاکستری', 'آبی', 'سبز کله‌غازی'], ARRAY['مشکی-آبی دو رنگ (طرح خاص BYD)'], '/catalog/cars/byd-atto-3.jpg'),
    ('کی‌ام‌سی EJ7', 'KMC EJ7 (Sehol E50A)', 'JAC/KMC', 'سدان', 'C', 'Sedan', 'اقتصادی', 'چین', 'کرمان موتور', 'JAC e-Platform', 50.1, '400V', 47, 'CATL', 'LFP', 'مایع‌خنک', 402, 'NEDC', 13.8, 142, 340, 1, 'PMSM', 7.6, 142, 'FWD', 6.6, 'GB/T', 62, 'GB/T', '30-80%', 48, 'خیر', 'خیر', 'خیر', 'L2', 1, 1, 1650, 540, 'کرمان موتور', ARRAY['EJ7: فقط مشکی متالیک | EJ7 پلاس: سفید', 'مشکی متالیک', 'خاکستری (بخشنامه کرمان موتور)'], ARRAY['مشکی (پلاس: مشکی-قهوه‌ای)'], '/catalog/cars/kmc-ej7-sehol-e50a.jpg'),
    ('هونگچی E-QM5', 'Hongqi E-QM5', 'Hongqi', 'سدان', 'D', 'Sedan', 'میان‌رده', 'چین', 'گروه بهمن', 'FME EV Platform', 55.66, '347V', 53, 'CATL', 'NMC', 'مایع‌خنک', 431, 'NEDC', 13.5, 100, 260, 1, 'PMSM', 10, 155, 'FWD', 6.6, 'GB/T', 60, 'GB/T', '20-80%', 40, 'خیر', 'خیر', 'خیر', 'L2', 1, 1, 1870, 446, 'تاکسی و شخصی', ARRAY['سفید', 'مشکی', 'خاکستری'], ARRAY['مشکی', 'بژ'], '/catalog/cars/hongqi-e-qm5.jpg'),
    ('لونا GRE', 'Luna GRE', 'MAPLE', 'سدان', 'C', 'Sedan', 'اقتصادی', 'چین', 'ایران‌خودرو', 'Geely GE (Maple 60S)', 43.9, '400V', 41, 'EVE', 'LFP', 'مایع‌خنک', 410, 'NEDC', 12, 110, 220, 1, 'PMSM', 9.9, 150, 'FWD', 6.6, 'GB/T', 50, 'GB/T', '30-80%', 30, 'خیر', 'خیر', 'خیر', 'L1', 1, 1, 1400, 410, 'ایران خودرو', ARRAY['فقط سفید (تمام واحدهای عرضه‌شده ایران‌خودرو)'], ARRAY['مشکی-خاکستری'], '/catalog/cars/luna-gre.jpg'),
    ('تویوتا بی‌زد3', 'Toyota bZ3 50kWh', 'Toyota', 'سدان', 'D', 'Sedan', 'میان‌رده', 'چین', 'معین خودرو / سایر واردکنندگان', 'e-TNGA', 49.9, '400V', 48, 'BYD', 'LFP Blade', 'مایع‌خنک', 517, 'CLTC', 11.5, 135, 303, 1, 'PMSM', 8, 190, 'RWD', 7, 'GB/T', 90, 'GB/T', '10-80%', 30, 'خیر', 'خیر', 'بله', 'L2', 1, 1, 1710, 439, 'BYD Battery', ARRAY['سفید', 'نقره‌ای', 'مشکی', 'خاکستری'], ARRAY['مشکی', 'دو رنگ روشن'], '/catalog/cars/toyota-bz3-50kwh.jpg'),
    ('بستیون NAT', 'Bestune NAT', 'FAW Bestune', 'MPV', 'MPV', 'MPV', 'اقتصادی', 'چین', 'BM Cars / گروه بهمن', 'FME', 55, '400V', 52, 'CATL', 'LFP', 'مایع‌خنک', 425, 'NEDC', 13, 120, 155, 1, 'PMSM', 10.8, 140, 'FWD', 6.6, 'GB/T', 80, 'GB/T', '30-80%', 30, 'خیر', 'خیر', 'خیر', 'L1', 0, 1, 1700, 454, 'تاکسی برقی', ARRAY['سفید (ناوگان تاکسی/سازمانی)'], ARRAY['خاکستری-مشکی'], '/catalog/cars/bestune-nat.jpg'),
    ('دانگ فنگ E70', 'Dongfeng E70 EV 2023', 'Dongfeng', 'سدان', 'C', 'Sedan', 'اقتصادی', 'چین', 'ایران‌خودرو', 'Aeolus E70 Platform', 47.5, '400V', 45, 'CATL', 'LFP', 'مایع‌خنک', 401, 'CLTC', 13.2, 110, 210, 1, 'PMSM', 9.5, 150, 'FWD', 6.6, 'GB/T', 66, 'GB/T', '10-80%', 50, 'خیر', 'خیر', 'خیر', 'Basic', 0, 1, 1550, 502, 'تاکسی/سازمانی ایران‌خودرو', ARRAY['سفید (عرضه ایران‌خودرو)'], ARRAY['مشکی', 'مشکی-قهوه‌ای'], '/catalog/cars/dongfeng-e70-ev-2023.jpg'),
    ('آواتار ۱۱', 'Avatr 11', 'Avatr', 'کراس‌اوور', 'E', 'Crossover/SUV', 'لوکس', 'چین', 'گروه بهمن / BM Cars', 'CHN (Changan/Huawei/CATL)', 116.79, '750V', 116.79, 'CATL', 'NMC', 'مایع‌خنک + Heat Pump', 680, 'CLTC', 18, 425, 650, 2, 'PMSM', 4.5, 200, 'AWD', 11, 'GB/T', 240, 'GB/T', '30-80%', 25, 'بله', 'خیر', 'بله', 'L2+', 3, 4, 2310, 469, 'نسخه دوموتوره 116kWh وارداتی بهمن/BM Cars؛ نسخه چینی 90.38kWh هم وجود دارد', ARRAY['خاکستری', 'نقره‌ای', 'سفید', 'مشکی (رنگ کارامل و مات به ایران نیامده)'], ARRAY['چرم سیاه', 'سفید', 'قهوه‌ای یا قرمز - انتخاب تریم نهایی با کارخانه'], '/catalog/cars/avatr-11.jpg'),
    ('بی‌ام‌و i3 برقی 35L', 'BMW i3 eDrive 35L', 'BMW', 'سدان', 'D', 'Sedan', 'لوکس', 'چین', 'پرشیا خودرو', 'CLAR (EV)', 70.3, '400V', 66.09999999999999, 'CATL', 'NMC', 'مایع‌خنک + Heat Pump', 526, 'CLTC', 14.3, 210, 400, 1, 'EESM (BMW Gen5)', 6.2, 180, 'RWD', 11, 'GB/T', 95, 'GB/T', '10-80%', 35, 'خیر', 'خیر', 'بله', 'L2', 1, 1, 2029, 410, 'مونتاژ BMW Brilliance چین؛ واردات پرشیا خودرو', ARRAY['سفید آلپاین', 'مشکی', 'خاکستری', 'آبی'], ARRAY['چرم Sensatec مشکی', 'کرم Oyster'], '/catalog/cars/bmw-i3-edrive-35l.jpg'),
    ('بی‌ام‌و i3 برقی 40L', 'BMW i3 eDrive 40L', 'BMW', 'سدان', 'D', 'Sedan', 'لوکس', 'چین', 'پرشیا خودرو', 'CLAR (EV)', 79.05, '400V', 74.40000000000001, 'CATL', 'NMC', 'مایع‌خنک + Heat Pump', 592, 'CLTC', 14.7, 250, 430, 1, 'EESM (BMW Gen5)', 5.6, 180, 'RWD', 11, 'GB/T', 95, 'GB/T', '10-80%', 35, 'خیر', 'خیر', 'بله', 'L2', 1, 1, 2080, 410, 'نسخه قوی‌تر i3؛ گرم‌کن صندلی و شارژر وایرلس بیشتر از 35L', ARRAY['سفید آلپاین', 'مشکی', 'خاکستری', 'آبی'], ARRAY['چرم Sensatec مشکی', 'کرم Oyster'], '/catalog/cars/bmw-i3-edrive-40l.jpg'),
    ('بی‌ام‌و iX3', 'BMW iX3', 'BMW', 'شاسی‌بلند', 'D', 'SUV', 'لوکس', 'چین', 'پرشیا خودرو / ژوبین', 'CLAR (EV)', 80, '400V', 74, 'CATL', 'NMC', 'مایع‌خنک + Heat Pump', 460, 'WLTP', 18.5, 210, 400, 1, 'EESM (BMW Gen5)', 6.8, 180, 'RWD', 11, 'GB/T (چین) / CCS2 (خلیج)', 150, 'GB/T (چین) / CCS2 (خلیج)', '10-80%', 32, 'خیر', 'خیر', 'بله', 'L2', 1, 3, 2185, 510, 'G08 ساخت چین؛ نام مدل اصلاح شد (قبلاً فقط BMW نوشته شده بود)', ARRAY['مشکی', 'خاکستری', 'سفید', 'آبی', 'قرمز', 'نقره‌ای (واردات پرشیا خودرو)'], ARRAY['چرم مشکی', 'کرم Oyster', 'قهوه‌ای Cognac'], '/catalog/cars/bmw-ix3.jpg'),
    ('کی‌ام‌سی G7', 'KMC G7 (Sehol A5 EV)', 'JAC/KMC', 'سدان', 'C', 'Sedan', 'اقتصادی/میان‌رده', 'چین', 'کرمان موتور', 'JAC e-Platform', 50.1, '400V', 47, 'Gotion/CATL', 'LFP', 'مایع‌خنک', 402, 'NEDC', 13.5, 142, 340, 1, 'PMSM', 7.6, 150, 'FWD', 6.6, 'GB/T', 62, 'GB/T', '30-80%', 48, 'خیر', 'خیر', 'خیر', 'L2', 1, 1, 1670, 540, 'فست‌بک هم‌پلتفرم EJ7؛ مونتاژ کرمان موتور', ARRAY['سفید', 'مشکی', 'خاکستری (مشابه EJ7)'], ARRAY['مشکی'], '/catalog/cars/kmc-g7-sehol-a5-ev.jpg'),
    ('سینوگلد تانگو 5', 'Sinogold Tango 5', 'Sinogold', 'سدان', 'C', 'Sedan', 'اقتصادی', 'چین', '(عمدتاً ناوگان تاکسی)', 'Arrizo 5e-based', 53.6, '400V', 50, 'CATL', 'LFP', 'مایع‌خنک', 401, 'NEDC', 13.5, 120, 250, 1, 'PMSM', 9, 150, 'FWD', 6.6, 'GB/T', 60, 'GB/T', '30-80%', 45, 'خیر', 'خیر', 'خیر', 'Basic', 0, 1, 1560, 430, 'بر پایه آریزو 5e؛ عمدتاً ناوگان تاکسی - اکثر اعداد نیاز به تأیید رسمی دارند', ARRAY['سفید (ناوگان تاکسی)'], ARRAY['مشکی-خاکستری'], '/catalog/cars/sinogold-tango-5.jpg'),
    ('آی‌ام LS7', 'IM LS7', 'IM Motors', 'شاسی‌بلند', 'E', 'SUV', 'لوکس', 'چین', 'نیکا موتور', 'IM platform (SAIC)', 100, '400V', 93, 'CATL', 'NMC', 'مایع‌خنک + Heat Pump', 611, 'WLTP/GCC', 18.5, 425, 725, 2, 'PMSM', 4.5, 200, 'AWD', 11, 'Type 2', 200, 'CCS2', '30-80%', 20, 'بله', 'خیر', 'بله', 'L2+', 5, 11, 2480, 733, 'نسخه GCC دوموتوره ~570hp؛ واردات نیکا موتور (عرضه از اسفند 1404/فروردین 1405)', ARRAY['سفید', 'مشکی', 'کرم', 'خاکستری (بخشنامه نیکا موتور)'], ARRAY['سفید-خاکستری', 'مشکی', 'قهوه‌ای'], '/catalog/cars/im-ls7.jpg');

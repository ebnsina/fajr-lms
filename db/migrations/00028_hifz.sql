-- +goose Up
-- +goose StatementBegin

-- Memorising the Qur'an: what a student has committed, and the daily record a
-- teacher keeps of new lesson, recent revision and old revision.
CREATE TYPE hifz_kind AS ENUM ('sabaq', 'sabqi', 'manzil');
CREATE TYPE hifz_quality AS ENUM ('excellent', 'good', 'fair', 'weak');

-- The surahs, with their real ayah counts, so a range can be checked rather
-- than trusted. Reference data: the same for every school, so it is not
-- tenant-scoped and nobody edits it.
CREATE TABLE surahs (
  number     smallint PRIMARY KEY CHECK (number BETWEEN 1 AND 114),
  name_ar    text NOT NULL,
  name_en    text NOT NULL,
  ayah_count smallint NOT NULL CHECK (ayah_count > 0)
);

INSERT INTO surahs (number, name_ar, name_en, ayah_count) VALUES
  (1, 'سُورَةُ ٱلْفَاتِحَةِ', 'Al-Faatiha', 7),
  (2, 'سُورَةُ البَقَرَةِ', 'Al-Baqara', 286),
  (3, 'سُورَةُ آلِ عِمۡرَانَ', 'Aal-i-Imraan', 200),
  (4, 'سُورَةُ النِّسَاءِ', 'An-Nisaa', 176),
  (5, 'سُورَةُ المَائـِدَةِ', 'Al-Maaida', 120),
  (6, 'سُورَةُ الأَنۡعَامِ', 'Al-An''aam', 165),
  (7, 'سُورَةُ الأَعۡرَافِ', 'Al-A''raaf', 206),
  (8, 'سُورَةُ الأَنفَالِ', 'Al-Anfaal', 75),
  (9, 'سُورَةُ التَّوۡبَةِ', 'At-Tawba', 129),
  (10, 'سُورَةُ يُونُسَ', 'Yunus', 109),
  (11, 'سُورَةُ هُودٍ', 'Hud', 123),
  (12, 'سُورَةُ يُوسُفَ', 'Yusuf', 111),
  (13, 'سُورَةُ الرَّعۡدِ', 'Ar-Ra''d', 43),
  (14, 'سُورَةُ إِبۡرَاهِيمَ', 'Ibrahim', 52),
  (15, 'سُورَةُ الحِجۡرِ', 'Al-Hijr', 99),
  (16, 'سُورَةُ النَّحۡلِ', 'An-Nahl', 128),
  (17, 'سُورَةُ الإِسۡرَاءِ', 'Al-Israa', 111),
  (18, 'سُورَةُ الكَهۡفِ', 'Al-Kahf', 110),
  (19, 'سُورَةُ مَرۡيَمَ', 'Maryam', 98),
  (20, 'سُورَةُ طه', 'Taa-Haa', 135),
  (21, 'سُورَةُ الأَنبِيَاءِ', 'Al-Anbiyaa', 112),
  (22, 'سُورَةُ الحَجِّ', 'Al-Hajj', 78),
  (23, 'سُورَةُ المُؤۡمِنُونَ', 'Al-Muminoon', 118),
  (24, 'سُورَةُ النُّورِ', 'An-Noor', 64),
  (25, 'سُورَةُ الفُرۡقَانِ', 'Al-Furqaan', 77),
  (26, 'سُورَةُ الشُّعَرَاءِ', 'Ash-Shu''araa', 227),
  (27, 'سُورَةُ النَّمۡلِ', 'An-Naml', 93),
  (28, 'سُورَةُ القَصَصِ', 'Al-Qasas', 88),
  (29, 'سُورَةُ العَنكَبُوتِ', 'Al-Ankaboot', 69),
  (30, 'سُورَةُ الرُّومِ', 'Ar-Room', 60),
  (31, 'سُورَةُ لُقۡمَانَ', 'Luqman', 34),
  (32, 'سُورَةُ السَّجۡدَةِ', 'As-Sajda', 30),
  (33, 'سُورَةُ الأَحۡزَابِ', 'Al-Ahzaab', 73),
  (34, 'سُورَةُ سَبَإٍ', 'Saba', 54),
  (35, 'سُورَةُ فَاطِرٍ', 'Faatir', 45),
  (36, 'سُورَةُ يسٓ', 'Yaseen', 83),
  (37, 'سُورَةُ الصَّافَّاتِ', 'As-Saaffaat', 182),
  (38, 'سُورَةُ صٓ', 'Saad', 88),
  (39, 'سُورَةُ الزُّمَرِ', 'Az-Zumar', 75),
  (40, 'سُورَةُ غَافِرٍ', 'Ghafir', 85),
  (41, 'سُورَةُ فُصِّلَتۡ', 'Fussilat', 54),
  (42, 'سُورَةُ الشُّورَىٰ', 'Ash-Shura', 53),
  (43, 'سُورَةُ الزُّخۡرُفِ', 'Az-Zukhruf', 89),
  (44, 'سُورَةُ الدُّخَانِ', 'Ad-Dukhaan', 59),
  (45, 'سُورَةُ الجَاثِيَةِ', 'Al-Jaathiya', 37),
  (46, 'سُورَةُ الأَحۡقَافِ', 'Al-Ahqaf', 35),
  (47, 'سُورَةُ مُحَمَّدٍ', 'Muhammad', 38),
  (48, 'سُورَةُ الفَتۡحِ', 'Al-Fath', 29),
  (49, 'سُورَةُ الحُجُرَاتِ', 'Al-Hujuraat', 18),
  (50, 'سُورَةُ قٓ', 'Qaaf', 45),
  (51, 'سُورَةُ الذَّارِيَاتِ', 'Adh-Dhaariyat', 60),
  (52, 'سُورَةُ الطُّورِ', 'At-Tur', 49),
  (53, 'سُورَةُ النَّجۡمِ', 'An-Najm', 62),
  (54, 'سُورَةُ القَمَرِ', 'Al-Qamar', 55),
  (55, 'سُورَةُ الرَّحۡمَٰن', 'Ar-Rahmaan', 78),
  (56, 'سُورَةُ الوَاقِعَةِ', 'Al-Waaqia', 96),
  (57, 'سُورَةُ الحَدِيدِ', 'Al-Hadid', 29),
  (58, 'سُورَةُ المُجَادلَةِ', 'Al-Mujaadila', 22),
  (59, 'سُورَةُ الحَشۡرِ', 'Al-Hashr', 24),
  (60, 'سُورَةُ المُمۡتَحنَةِ', 'Al-Mumtahana', 13),
  (61, 'سُورَةُ الصَّفِّ', 'As-Saff', 14),
  (62, 'سُورَةُ الجُمُعَةِ', 'Al-Jumu''a', 11),
  (63, 'سُورَةُ المُنَافِقُونَ', 'Al-Munaafiqoon', 11),
  (64, 'سُورَةُ التَّغَابُنِ', 'At-Taghaabun', 18),
  (65, 'سُورَةُ الطَّلَاقِ', 'At-Talaaq', 12),
  (66, 'سُورَةُ التَّحۡرِيمِ', 'At-Tahrim', 12),
  (67, 'سُورَةُ المُلۡكِ', 'Al-Mulk', 30),
  (68, 'سُورَةُ القَلَمِ', 'Al-Qalam', 52),
  (69, 'سُورَةُ الحَاقَّةِ', 'Al-Haaqqa', 52),
  (70, 'سُورَةُ المَعَارِجِ', 'Al-Ma''aarij', 44),
  (71, 'سُورَةُ نُوحٍ', 'Nooh', 28),
  (72, 'سُورَةُ الجِنِّ', 'Al-Jinn', 28),
  (73, 'سُورَةُ المُزَّمِّلِ', 'Al-Muzzammil', 20),
  (74, 'سُورَةُ المُدَّثِّرِ', 'Al-Muddaththir', 56),
  (75, 'سُورَةُ القِيَامَةِ', 'Al-Qiyaama', 40),
  (76, 'سُورَةُ الإِنسَانِ', 'Al-Insaan', 31),
  (77, 'سُورَةُ المُرۡسَلَاتِ', 'Al-Mursalaat', 50),
  (78, 'سُورَةُ النَّبَإِ', 'An-Naba', 40),
  (79, 'سُورَةُ النَّازِعَاتِ', 'An-Naazi''aat', 46),
  (80, 'سُورَةُ عَبَسَ', 'Abasa', 42),
  (81, 'سُورَةُ التَّكۡوِيرِ', 'At-Takwir', 29),
  (82, 'سُورَةُ الانفِطَارِ', 'Al-Infitaar', 19),
  (83, 'سُورَةُ المُطَفِّفِينَ', 'Al-Mutaffifin', 36),
  (84, 'سُورَةُ الانشِقَاقِ', 'Al-Inshiqaaq', 25),
  (85, 'سُورَةُ البُرُوجِ', 'Al-Burooj', 22),
  (86, 'سُورَةُ الطَّارِقِ', 'At-Taariq', 17),
  (87, 'سُورَةُ الأَعۡلَىٰ', 'Al-A''laa', 19),
  (88, 'سُورَةُ الغَاشِيَةِ', 'Al-Ghaashiya', 26),
  (89, 'سُورَةُ الفَجۡرِ', 'Al-Fajr', 30),
  (90, 'سُورَةُ البَلَدِ', 'Al-Balad', 20),
  (91, 'سُورَةُ الشَّمۡسِ', 'Ash-Shams', 15),
  (92, 'سُورَةُ اللَّيۡلِ', 'Al-Lail', 21),
  (93, 'سُورَةُ الضُّحَىٰ', 'Ad-Dhuhaa', 11),
  (94, 'سُورَةُ الشَّرۡحِ', 'Ash-Sharh', 8),
  (95, 'سُورَةُ التِّينِ', 'At-Tin', 8),
  (96, 'سُورَةُ العَلَقِ', 'Al-Alaq', 19),
  (97, 'سُورَةُ القَدۡرِ', 'Al-Qadr', 5),
  (98, 'سُورَةُ البَيِّنَةِ', 'Al-Bayyina', 8),
  (99, 'سُورَةُ الزَّلۡزَلَةِ', 'Az-Zalzala', 8),
  (100, 'سُورَةُ العَادِيَاتِ', 'Al-Aadiyaat', 11),
  (101, 'سُورَةُ القَارِعَةِ', 'Al-Qaari''a', 11),
  (102, 'سُورَةُ التَّكَاثُرِ', 'At-Takaathur', 8),
  (103, 'سُورَةُ العَصۡرِ', 'Al-Asr', 3),
  (104, 'سُورَةُ الهُمَزَةِ', 'Al-Humaza', 9),
  (105, 'سُورَةُ الفِيلِ', 'Al-Fil', 5),
  (106, 'سُورَةُ قُرَيۡشٍ', 'Quraish', 4),
  (107, 'سُورَةُ المَاعُونِ', 'Al-Maa''un', 7),
  (108, 'سُورَةُ الكَوۡثَرِ', 'Al-Kawthar', 3),
  (109, 'سُورَةُ الكَافِرُونَ', 'Al-Kaafiroon', 6),
  (110, 'سُورَةُ النَّصۡرِ', 'An-Nasr', 3),
  (111, 'سُورَةُ المَسَدِ', 'Al-Masad', 5),
  (112, 'سُورَةُ الإِخۡلَاصِ', 'Al-Ikhlaas', 4),
  (113, 'سُورَةُ الفَلَقِ', 'Al-Falaq', 5),
  (114, 'سُورَةُ النَّاسِ', 'An-Naas', 6);

-- One sitting with a teacher. Sabaq is the new lesson, sabqi the recent
-- revision, manzil the older revision a student keeps warm.
CREATE TABLE hifz_entries (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  student_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  teacher_id  uuid REFERENCES users(id) ON DELETE SET NULL,
  on_date     date NOT NULL DEFAULT current_date,
  kind        hifz_kind NOT NULL,
  from_surah  smallint NOT NULL REFERENCES surahs(number),
  from_ayah   smallint NOT NULL CHECK (from_ayah > 0),
  to_surah    smallint NOT NULL REFERENCES surahs(number),
  to_ayah     smallint NOT NULL CHECK (to_ayah > 0),
  quality     hifz_quality NOT NULL DEFAULT 'good',
  mistakes    smallint NOT NULL DEFAULT 0 CHECK (mistakes >= 0 AND mistakes <= 999),
  note        text NOT NULL DEFAULT '' CHECK (length(note) <= 500),
  created_at  timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT a_range_runs_forwards CHECK (
    to_surah > from_surah OR (to_surah = from_surah AND to_ayah >= from_ayah)
  )
);

CREATE INDEX hifz_entries_student_idx ON hifz_entries (tenant_id, student_id, on_date DESC);
CREATE INDEX hifz_entries_date_idx ON hifz_entries (tenant_id, on_date DESC);

ALTER TABLE hifz_entries ENABLE ROW LEVEL SECURITY;
ALTER TABLE hifz_entries FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON hifz_entries
  USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id());

GRANT SELECT, INSERT, UPDATE, DELETE ON hifz_entries TO fajr_app;
GRANT SELECT ON surahs TO fajr_app;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS hifz_entries;
DROP TABLE IF EXISTS surahs;
DROP TYPE IF EXISTS hifz_quality;
DROP TYPE IF EXISTS hifz_kind;
-- +goose StatementEnd

-- +goose Up
-- +goose StatementBegin

-- What kind of place this is. A madrasah and a coaching centre want different
-- words for the same rows, and later different report cards.
CREATE TYPE institution_kind AS ENUM ('school', 'college', 'madrasah', 'coaching', 'other');

ALTER TABLE tenants
  ADD COLUMN institution institution_kind NOT NULL DEFAULT 'school';

-- The year a school teaches in, and the terms inside it. Exactly one of each
-- is current, which is what every other screen reads without being told.
CREATE TABLE academic_years (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name       text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 60),
  starts_on  date NOT NULL,
  ends_on    date NOT NULL,
  is_current boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, name),
  CONSTRAINT a_year_ends_after_it_starts CHECK (ends_on > starts_on)
);

CREATE UNIQUE INDEX one_current_year ON academic_years (tenant_id) WHERE is_current;

CREATE TABLE terms (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  year_id    uuid NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
  name       text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 60),
  starts_on  date NOT NULL,
  ends_on    date NOT NULL,
  is_current boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (year_id, name),
  CONSTRAINT a_term_ends_after_it_starts CHECK (ends_on > starts_on)
);

CREATE UNIQUE INDEX one_current_term ON terms (tenant_id) WHERE is_current;

-- The ladder a school types in itself: Class Six, Hifz, Ibtidaiyyah, HSC 1st
-- Year. We ship no ladder of our own, because nobody's matches anybody else's.
CREATE TABLE classes (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name       text NOT NULL COLLATE human_name CHECK (length(btrim(name)) BETWEEN 1 AND 100),
  rank       integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, name)
);

CREATE TABLE sections (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  class_id   uuid NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
  name       text NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 60),
  capacity   integer CHECK (capacity IS NULL OR capacity > 0),
  teacher_id uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (class_id, name)
);

CREATE TABLE subjects (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  class_id   uuid REFERENCES classes(id) ON DELETE CASCADE,
  name       text NOT NULL COLLATE human_name CHECK (length(btrim(name)) BETWEEN 1 AND 100),
  code       text NOT NULL DEFAULT '' CHECK (length(code) <= 30),
  dir        text_dir NOT NULL DEFAULT 'auto',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX subjects_named_once_per_class ON subjects (tenant_id, class_id, name)
  WHERE class_id IS NOT NULL;
CREATE UNIQUE INDEX subjects_named_once_school_wide ON subjects (tenant_id, name)
  WHERE class_id IS NULL;

-- Where a student sits this year. A person can be in one section per year, and
-- the roll is theirs within it.
CREATE TABLE placements (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  year_id    uuid NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
  section_id uuid NOT NULL REFERENCES sections(id) ON DELETE CASCADE,
  user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  roll_no    integer CHECK (roll_no IS NULL OR roll_no > 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (year_id, user_id)
);

CREATE UNIQUE INDEX one_roll_per_section ON placements (section_id, roll_no)
  WHERE roll_no IS NOT NULL;
CREATE INDEX placements_section_idx ON placements (section_id);
CREATE INDEX sections_class_idx ON sections (class_id);
CREATE INDEX terms_year_idx ON terms (year_id);

-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['academic_years', 'terms', 'classes', 'sections', 'subjects', 'placements'] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (tenant_id = current_tenant_id()) WITH CHECK (tenant_id = current_tenant_id())', t);
    EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON %I TO fajr_app', t);
  END LOOP;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER academic_years_touch BEFORE UPDATE ON academic_years FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER terms_touch BEFORE UPDATE ON terms FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER classes_touch BEFORE UPDATE ON classes FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER sections_touch BEFORE UPDATE ON sections FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER subjects_touch BEFORE UPDATE ON subjects FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS placements;
DROP TABLE IF EXISTS subjects;
DROP TABLE IF EXISTS sections;
DROP TABLE IF EXISTS classes;
DROP TABLE IF EXISTS terms;
DROP TABLE IF EXISTS academic_years;
ALTER TABLE tenants DROP COLUMN IF EXISTS institution;
DROP TYPE IF EXISTS institution_kind;
-- +goose StatementEnd

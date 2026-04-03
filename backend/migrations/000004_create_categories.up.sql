-- CREATE CATEGORIES TABLE

CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    parent_id UUID REFERENCES categories(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT,
    icon_url TEXT,
    display_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Length Constraints
    CONSTRAINT chk_name_length CHECK (char_length(name) >= 3 AND char_length(name) <= 100),
    CONSTRAINT chk_slug_length CHECK (char_length(slug) >= 3 AND char_length(slug) <= 100),
    CONSTRAINT chk_description_length CHECK (description IS NULL OR char_length(description) <= 1000),
    CONSTRAINT chk_icon_url_length CHECK (icon_url IS NULL OR char_length(icon_url) <= 2048),
    -- Slug Format Validation
    CONSTRAINT chk_slug_format CHECK (slug ~* '[a-z0-9]+(-[a-z0-9]+)*$'),
    -- Parent-Child Relationship Validation
    CONSTRAINT chk_no_self_reference CHECK (parent_id IS NULL OR parent_id != id)
);

-- INDEXES
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);
CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_active_display_order ON categories(display_order) WHERE is_active = TRUE;

-- Trigger to call the function
-- DROP TRIGGER IF EXISTS update_categories_updated_at ON categories; (NOTE: suggested by qwen)
CREATE OR REPLACE TRIGGER update_categories_updated_at
BEFORE UPDATE ON categories
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
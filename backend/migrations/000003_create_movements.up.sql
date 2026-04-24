-- CREATE MOVEMENTS TABLE

-- ENUMS
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'movement_status') THEN
        CREATE TYPE movement_status AS ENUM ('draft', 'active', 'archived', 'rejected', 'pending_review');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS movements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    short_description TEXT NOT NULL,
    long_description TEXT,
    image_url TEXT,
    icon_url TEXT,
    website_url TEXT,
    claimed_by_org_id UUID,
    supporter_count INTEGER NOT NULL DEFAULT 0,
    trending_score NUMERIC(10,4) NOT NULL DEFAULT 0,
    status movement_status NOT NULL DEFAULT 'draft',
    created_by_user_id UUID,
    reviewed_by_user_id UUID,
    reviewed_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign Keys
    CONSTRAINT fk_movements_claimed_by_org_id FOREIGN KEY (claimed_by_org_id) REFERENCES organizations(id) ON DELETE SET NULL,
    CONSTRAINT fk_movements_created_by_user_id FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_movements_reviewed_by_user_id FOREIGN KEY (reviewed_by_user_id) REFERENCES users(id) ON DELETE SET NULL,

    -- Length Constraints
    CONSTRAINT chk_slug_length CHECK (char_length(slug) >= 3 AND char_length(slug) <= 100),
    CONSTRAINT chk_name_length CHECK (char_length(name) >= 3 AND char_length(name) <= 200),
    CONSTRAINT chk_short_description_length CHECK (char_length(short_description) >= 10 AND char_length(short_description) <= 500),
    CONSTRAINT chk_long_description_length CHECK (long_description IS NULL OR char_length(long_description) <= 5000),
    CONSTRAINT chk_image_url_length CHECK (image_url IS NULL OR char_length(image_url) <= 2048),
    CONSTRAINT chk_icon_url_length CHECK (icon_url IS NULL OR char_length(icon_url) <= 2048),
    CONSTRAINT chk_website_url_length CHECK (website_url IS NULL OR char_length(website_url) <= 2048),
    CONSTRAINT chk_supporter_count_non_negative CHECK (supporter_count >= 0),
    CONSTRAINT chk_trending_score_non_negative CHECK (trending_score >= 0),
    -- Slug Format Validation
    CONSTRAINT chk_slug_format CHECK (slug ~ '[a-z0-9]+(-[a-z0-9]+)*$')
);

-- Indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_movements_slug_unique ON movements(slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_movements_status ON movements(status);
CREATE INDEX IF NOT EXISTS idx_movements_name_trgm ON movements USING gin (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_supporter_count ON movements(supporter_count DESC) WHERE status = 'active' AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_trending_score ON movements(trending_score DESC) WHERE status = 'active' AND deleted_at IS NULL;
-- CREATE INDEX IF NOT EXISTS idx_movements_claimed_by_org_id ON movements(claimed_by_org_id);
CREATE INDEX IF NOT EXISTS idx_movements_created_at_active ON movements(created_at DESC) WHERE status = 'active' AND deleted_at IS NULL;

-- Trigger to call the function
CREATE OR REPLACE TRIGGER update_movements_updated_at
BEFORE UPDATE ON movements
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
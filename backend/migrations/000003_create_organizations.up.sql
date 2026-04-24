-- CREATE ORGANIZATIONS TABLE

-- Enums
DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'org_status') THEN
		CREATE TYPE org_status AS ENUM ('active', 'inactive', 'suspended', 'rejected');
	END IF;

	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'org_verification_status') THEN
		CREATE TYPE org_verification_status AS ENUM ('unverified', 'pending', 'verified', 'rejected');
	END IF;
END $$;


CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    short_description TEXT,
    long_description TEXT,
    logo_url TEXT,
    cover_image_url TEXT,
    website_url TEXT,
    contact_email TEXT,
    ein_tax_id_hash TEXT,
    country_code CHAR(2),
    status org_status NOT NULL DEFAULT 'inactive',
    verification_status org_verification_status NOT NULL DEFAULT 'unverified',
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    verified_by_user_id UUID,
    created_by_user_id UUID,
    social_links JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ

    -- Foreign Keys
    CONSTRAINT fk_organizations_verified_by FOREIGN KEY (verified_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_organizations_created_by FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE RESTRICT,

    -- Length Constraints
    CONSTRAINT chk_slug_length CHECK (char_length(slug) >= 3 AND char_length(slug) <= 100),
    CONSTRAINT chk_name_length CHECK (char_length(name) >= 3 AND char_length(name) <= 150),
    CONSTRAINT chk_short_description_length CHECK (short_description IS NULL OR char_length(short_description) <= 300),
    CONSTRAINT chk_long_description_length CHECK (long_description IS NULL OR char_length(long_description) <= 5000),
    CONSTRAINT chk_logo_url_length CHECK (logo_url IS NULL OR char_length(logo_url) <= 2048),

    CONSTRAINT chk_cover_image_url_length CHECK (cover_image_url IS NULL OR char_length(cover_image_url) <= 2048),
    CONSTRAINT chk_website_url_length CHECK (website_url IS NULL OR char_length(website_url) <= 2048),
    CONSTRAINT chk_contact_email_length CHECK (contact_email IS NULL OR char_length(contact_email) <= 255),

    -- Format Constraints
    CONSTRAINT chk_slug_format CHECK (slug ~ '^[a-z0-9-]+$'),
    CONSTRAINT chk_contact_email_format CHECK (contact_email IS NULL OR contact_email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT chk_country_code_format CHECK (country_code IS NULL OR country_code ~ '^[A-Z]{2}$'),
    CONSTRAINT chk_website_url_format CHECK (website_url IS NULL OR website_url ~* '^https?://'),

    -- Logical Consistency
    CONSTRAINT chk_verified_consistency
        CHECK (
            (is_verified = FALSE AND verified_at IS NULL AND verified_by_user_id IS NULL)
            OR
            (is_verified = TRUE AND verified_at IS NOT NULL AND verified_by_user_id IS NOT NULL)
        )
);

-- Indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug);
CREATE INDEX IF NOT EXISTS idx_organizations_status ON organizations(status);
CREATE INDEX IF NOT EXISTS idx_organizations_verification_status ON organizations(verification_status);
CREATE INDEX IF NOT EXISTS idx_organizations_country_code ON organizations(country_code);
CREATE INDEX IF NOT EXISTS idx_organizations_created_by_user_id ON organizations(created_by_user_id);
CREATE INDEX IF NOT EXISTS idx_organizations_created_at ON organizations(created_at);
CREATE INDEX IF NOT EXISTS idx_organizations_deleted_at ON organizations(deleted_at) WHERE deleted_at IS NULL;

-- GIN index for JSONB social_links queries
CREATE INDEX IF NOT EXISTS idx_organizations_social_links ON organizations USING GIN (social_links);

-- Trigger to call the function
CREATE OR REPLACE TRIGGER update_organizations_updated_at
BEFORE UPDATE ON organizations
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
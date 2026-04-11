-- CREATE USER TABLE

-- Enums
DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_profile_visibility') THEN
		CREATE TYPE user_profile_visibility AS ENUM ('public', 'private', 'unlisted');
	END IF;

	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
		CREATE TYPE user_role AS ENUM ('user', 'moderator', 'admin', 'superadmin');
	END IF;

	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_status') THEN
		CREATE TYPE user_status AS ENUM ('active', 'suspended', 'banned', 'deactivated');
	END IF;
END $$;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username TEXT NOT NULL,
    email TEXT NOT NULL,
    email_verified_at TIMESTAMPTZ,
    password_hash TEXT,
    display_name TEXT NOT NULL,
    bio TEXT,
    avatar_url TEXT,
    location TEXT,
    profile_visibility user_profile_visibility NOT NULL DEFAULT 'public',
    embed_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    role user_role NOT NULL DEFAULT 'user',
    status user_status NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),


    -- Length Constraints
    CONSTRAINT chk_username_length CHECK (char_length(username) >= 3 AND char_length(username) <= 44),
    CONSTRAINT chk_email_length CHECK (char_length(email) <= 255),
    CONSTRAINT chk_display_name_length CHECK (char_length(display_name) >= 3 AND char_length(display_name) <= 50),
    CONSTRAINT chk_avatar_url_length CHECK (avatar_url IS NULL OR char_length(avatar_url) <= 2048),
    CONSTRAINT chk_bio_length CHECK (bio IS NULL OR char_length(bio) <= 1000),
    CONSTRAINT chk_location_length CHECK (location IS NULL OR char_length(location) <= 100),
    -- Email Validation
    CONSTRAINT chk_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    -- Username Validation
    CONSTRAINT chk_username_format CHECK (username ~ '^[a-z0-9_-]+$')
);

-- Indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);

-- Trigger to call the function
CREATE OR REPLACE TRIGGER update_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
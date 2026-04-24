-- CREATE MOVEMENTS TABLE

-- ENUMS
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'advocacy_status') THEN
        CREATE TYPE advocacy_status AS ENUM ('active', 'paused', 'removed');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'badge_level') THEN
        CREATE TYPE badge_level AS ENUM ('bronze', 'silver', 'gold', 'platinum', 'diamond');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS user_movements (
    id                  UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID            NOT NULL,
    movement_id         UUID            NOT NULL,
    personal_statement  TEXT,
    verification_tier   SMALLINT        NOT NULL DEFAULT 0,
    badge_level         badge_level     NOT NULL DEFAULT 'bronze',
    display_order       SMALLINT        NOT NULL DEFAULT 0,
    is_pinned           BOOLEAN         NOT NULL DEFAULT FALSE,
    is_public           BOOLEAN         NOT NULL DEFAULT TRUE,
    status              advocacy_status NOT NULL DEFAULT 'active',
    supported_since     TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    removed_at          TIMESTAMPTZ,

    -- Foreign Keys
    CONSTRAINT fk_user_movements_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_movements_movement FOREIGN KEY (movement_id) REFERENCES movements(id) ON DELETE CASCADE,

    -- A user can only be linked to the same movement once
    CONSTRAINT uq_user_movements_user_movement UNIQUE (user_id, movement_id),

    -- Length Constraints
    CONSTRAINT chk_user_movements_personal_statement_length CHECK (personal_statement IS NULL OR char_length(personal_statement) <= 1000),

    -- Range Constraints
    CONSTRAINT chk_user_movements_verification_tier CHECK (verification_tier >= 0 AND verification_tier <= 5),
    CONSTRAINT chk_user_movements_display_order CHECK (display_order >= 0),

    -- Logical Consistency
    CONSTRAINT chk_user_movements_removed_consistency
        CHECK (
            (status = 'removed' AND removed_at IS NOT NULL)
            OR
            (status != 'removed' AND removed_at IS NULL)
        )
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_user_movements_user_id ON user_movements(user_id);
CREATE INDEX IF NOT EXISTS idx_user_movements_movement_id ON user_movements(movement_id);
CREATE INDEX IF NOT EXISTS idx_user_movements_status ON user_movements(status);
CREATE INDEX IF NOT EXISTS idx_user_movements_badge_level ON user_movements(badge_level);
CREATE INDEX IF NOT EXISTS idx_user_movements_supported_since ON user_movements(supported_since);

-- Fetch a user's public active advocacy list, ordered for profile display
CREATE INDEX IF NOT EXISTS idx_user_movements_profile_display
    ON user_movements(user_id, display_order, is_pinned)
    WHERE status = 'active' AND is_public = TRUE;
    
-- Trigger to call the function
CREATE OR REPLACE TRIGGER update_user_movements_updated_at
BEFORE UPDATE ON user_movements
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
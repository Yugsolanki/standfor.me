-- CREATE RESERVERD USERNAMES 

CREATE TABLE reserved_usernames (
    username TEXT PRIMARY KEY, 
    reserved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_at TIMESTAMPTZ, -- NULL if still reserved, otherwise the time it was released
    released_by UUID REFERENCES users(id), -- Admin who released the username, NULL if still reserved
    reason TEXT NOT NULL, -- e.g., "Offensive", "Trademark", "Impersonation", "Deleted_User" etc.

    -- Length Constraints
    CONSTRAINT chk_username_length CHECK (char_length(username) >= 3 AND char_length(username) <= 30),
    CONSTRAINT chk_reason_length CHECK (char_length(reason) <= 255),
    -- Logical Constraints
    CONSTRAINT chk_release_time CHECK (released_at IS NULL OR released_at > reserved_at),
    CONSTRAINT chk_release_consistency CHECK (
        (released_at IS NULL AND released_by IS NULL) OR
        (released_at IS NOT NULL AND released_by IS NOT NULL)
    ), -- Ensure that if a username is released, it must have a release time and an admin who released it
    -- Username Format Constraints
    CONSTRAINT chk_username_format  CHECK (username ~ '^[a-z0-9_-]+$')
);
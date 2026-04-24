-- CREATE MOVEMENT  CATEGORIES TABLE

CREATE TABLE IF NOT EXISTS movement_categories (
    movement_id UUID NOT NULL,
    category_id UUID NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Composite Primary Key
    PRIMARY KEY (movement_id, category_id),

    -- Foreign Keys
    CONSTRAINT fk_movement_categories_movement FOREIGN KEY (movement_id) REFERENCES movements(id) ON DELETE CASCADE,
    CONSTRAINT fk_movement_categories_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_movement_categories_movement_id ON movement_categories(movement_id);
CREATE INDEX IF NOT EXISTS idx_movement_categories_category_id ON movement_categories(category_id);

-- Only one primary category allowed per movement
CREATE UNIQUE INDEX IF NOT EXISTS idx_movement_categories_one_primary
    ON movement_categories(movement_id)
    WHERE is_primary = TRUE;
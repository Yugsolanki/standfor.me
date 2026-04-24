DROP TRIGGER IF EXISTS update_user_movements_updated_at ON users;

DROP TABLE IF EXISTS user_movements;

DROP TYPE IF EXISTS advocacy_status;
DROP TYPE IF EXISTS badge_level;
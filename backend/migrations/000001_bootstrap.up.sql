-- Bootstrap migration to set up necessary extensions and functions for the application.

-- Create necessary extensions for UUID generation and trigram indexing.
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Create a function to automatically update the 'updated_at' timestamp on row updates.
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';
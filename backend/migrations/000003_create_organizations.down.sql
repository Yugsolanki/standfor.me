DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;

DROP TABLE IF EXISTS organizations;

DROP TYPE IF EXISTS org_status;
DROP TYPE IF EXISTS org_verification_status;
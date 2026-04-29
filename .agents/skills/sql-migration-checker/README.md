# SQL Migration Checker Skill

A comprehensive static analysis tool for PostgreSQL migration files that detects vulnerabilities, anti-patterns, N+1 queries, naming inconsistencies, and schema design issues.

## Quick Start

### Check All Migrations

```bash
# Full audit with color output
make sql-check

# CI-friendly format (markdown-compatible)
make sql-check-ci

# Generate detailed audit report
make sql-audit-report
```

### Check Single File

```bash
# Check a specific migration
make sql-check-one FILE=000009_create_user_movements.up.sql

# Or use the script directly
./scripts/sql-check.sh --file 000009_create_user_movements.up.sql
```

## What It Checks

### 🔴 CRITICAL (Must Fix)
- SQL injection vulnerabilities in dynamic SQL
- Missing migration pairs (.down.sql files)

### 🟡 WARNING (Should Fix)
- Foreign keys without ON DELETE/UPDATE policies
- Missing indexes on foreign key columns
- Non-idempotent CREATE statements (missing IF NOT EXISTS, OR REPLACE)
- Multi-statement migrations without transactions
- Naming convention violations (PascalCase vs snake_case)
- N+1 query patterns from single-column indexes

### 🔵 INFO (Best Practices)
- Missing audit fields (created_at, updated_at, deleted_at)
- TEXT columns without length constraints
- Missing partial indexes for boolean/status columns
- Index naming convention suggestions
- Constraint naming suggestions

## Example Output

```
=== SQL Migration Checker ===

Analyzing: 000009_create_user_movements.up.sql
🟡 WARNING: 000009_create_user_movements.up.sql:15 - Table name uses PascalCase, consider snake_case for consistency
🔵 INFO: 000009_create_user_movements.up.sql - Consider adding deleted_at for soft delete support

=== Summary ===

Files Analyzed:    18 (9 up, 9 down)

🔴 CRITICAL: 0
🟡 WARNING:  19
🔵 INFO:     35

⚠️  SQL check PASSED with 19 warning(s)
```

## Integration

### Pre-commit Hook

Add to `.git/hooks/pre-commit`:

```bash
#!/bin/bash
FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.sql$')
if [ -n "$FILES" ]; then
    echo "Running SQL migration check..."
    for file in $FILES; do
        ./scripts/sql-check.sh --file $(basename $file) || exit 1
    done
fi
```

### CI/CD Pipeline

```yaml
# Example GitHub Actions
- name: SQL Migration Check
  run: make sql-check-ci
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MIGRATIONS_DIR` | Path to migrations directory | `backend/migrations` |

## Command Line Options

```
Usage: ./scripts/sql-check.sh [OPTIONS]

Options:
  --file FILE   Check a single migration file
  --ci          CI-friendly output format (Markdown)
  --report      Generate detailed audit report (sql-migration-audit-report.md)
  --help        Show help message
```

## Detected Issues

### N+1 Query Patterns
```sql
-- ❌ Single-column index when queries filter on multiple columns
CREATE INDEX idx_user_movements_user_id ON user_movements(user_id);

-- Common query that would cause N+1:
SELECT * FROM user_movements WHERE user_id = $1 AND status = 'active';

-- ✅ Better: Composite index
CREATE INDEX idx_user_movements_user_status ON user_movements(user_id, status);
```

### Foreign Key Safety
```sql
-- ❌ Missing ON DELETE policy
FOREIGN KEY (user_id) REFERENCES users(id);

-- ✅ Explicit cascade policy
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE;
```

### Idempotency
```sql
-- ❌ Not safe to run multiple times
CREATE TYPE user_role AS ENUM ('user', 'admin');
CREATE TRIGGER update_updated_at BEFORE UPDATE ON users ...

-- ✅ Idempotent
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
        CREATE TYPE user_role AS ENUM ('user', 'admin');
    END IF;
END $$;

CREATE OR REPLACE TRIGGER update_users_updated_at ...
```

## License

MIT

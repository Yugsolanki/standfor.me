---
name: sql-migration-checker
description: Perform strict SQL migration audits for PostgreSQL .sql files, detecting vulnerabilities, anti-patterns, N+1 queries, naming inconsistencies, and schema design issues
license: MIT
compatibility: opencode
metadata:
  language: sql
  database: postgresql
  focus: migrations
  scope: static-analysis
  severity_levels: critical,warning,info
---

## What I do
- Perform **strict static analysis** on PostgreSQL `.sql` migration files
- Detect **N+1 query patterns** and inefficient query structures
- Identify **naming convention violations** (inconsistent casing, unclear identifiers)
- Check for **missing indexes** on foreign keys and frequently filtered columns
- Validate **foreign key constraints** (missing ON DELETE/UPDATE policies)
- Detect **SQL injection vulnerabilities** in dynamic SQL patterns
- Check for **missing NULL handling** and improper default values
- Identify **missing transaction safety** in multi-statement migrations
- Validate **enum usage consistency** and type safety
- Check for **potential deadlocks** in trigger/lock patterns
- Detect **schema design anti-patterns** (lack of soft delete, missing audit fields)
- Verify **index naming conventions** and redundant indexes
- Check for **spelling mistakes** in identifiers and comments

## When to use me
- Before committing new `.sql` migration files to `backend/migrations/`
- During CI/CD pipeline to block problematic migrations from deployment
- When auditing existing migrations for technical debt
- During code reviews of database schema changes
- Before running `make migrate-up` in production environments
- When refactoring or consolidating existing migrations

## Execution Guidelines (How I work)

1. **File Discovery:** 
   Scan all `.up.sql` and `.down.sql` files in `backend/migrations/` directory. Process paired files together to ensure rollback safety.

2. **Check Categories:** 
   Organize findings into three severity levels:
   - **CRITICAL**: Security vulnerabilities, data loss risks, schema-breaking changes
   - **WARNING**: Performance issues, missing indexes, naming inconsistencies
   - **INFO**: Best practice suggestions, minor optimizations

3. **Analysis Pattern:**

```markdown
### Migration: 000009_create_user_movements.up.sql

#### ✅ Passed Checks
- Foreign keys have ON DELETE CASCADE policies
- Enum types use IF NOT EXISTS guards
- Indexes follow naming convention (idx_table_column)
- Audit fields (created_at, updated_at) present

#### ⚠️ Warnings (2)
1. **N+1 Query Risk**: No composite index on (user_id, status) for common filter pattern
   - Location: Line 47
   - Suggestion: CREATE INDEX idx_user_movements_user_status ON user_movements(user_id, status);

2. **Naming Inconsistency**: Mixed snake_case and SCREAMING_SNAKE_CASE in comments
   - Location: Line 3, Line 23
   - Suggestion: Use consistent snake_case for all identifiers and comments

#### ℹ️ Info (1)
1. Consider adding deleted_at for soft delete capability (currently using status='removed')
```

4. **Check Categories I Perform:**

## A. Security Vulnerabilities (CRITICAL)

### SQL Injection Patterns
```sql
-- ❌ BAD: Dynamic table names without parameterization
EXECUTE format('CREATE TABLE %I', user_input);

-- ❌ BAD: String concatenation in EXECUTE
EXECUTE 'SELECT * FROM ' || table_name;
```

### Missing Access Controls
```sql
-- ❌ BAD: Creating functions without SECURITY DEFINER consideration
CREATE FUNCTION sensitive_operation() RETURNS void AS $$ ... $$;
```

## B. N+1 Query Patterns (WARNING)

### Missing Composite Indexes
```sql
-- ❌ BAD: Queries filtering on multiple columns without composite index
SELECT * FROM user_movements WHERE user_id = $1 AND status = $2;
-- Missing: CREATE INDEX idx_user_movements_user_status ...

-- ✅ GOOD: Composite index matches query pattern
CREATE INDEX idx_user_movements_user_status ON user_movements(user_id, status);
```

### Foreign Key Indexes
```sql
-- ❌ BAD: Foreign key without index (causes slow JOINs)
ALTER TABLE user_movements ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id);
-- Missing: CREATE INDEX idx_user_movements_user_id ...

-- ✅ GOOD: Index on FK column
CREATE INDEX idx_user_movements_user_id ON user_movements(user_id);
```

## C. Naming Conventions (WARNING/INFO)

### Identifier Consistency
```sql
-- ❌ BAD: Mixed naming styles
CREATE TABLE UserMovements ( ... );  -- PascalCase
CREATE TABLE user_movements ( ... ); -- snake_case

-- ✅ GOOD: Consistent snake_case
CREATE TABLE user_movements ( ... );
CREATE INDEX idx_user_movements_user_id ON user_movements(user_id);
```

### Index Naming
```sql
-- ❌ BAD: Unclear index names
CREATE INDEX idx_user ON user_movements(user_id);

-- ✅ GOOD: Descriptive index names
CREATE INDEX idx_user_movements_user_id ON user_movements(user_id);
```

### Constraint Naming
```sql
-- ❌ BAD: Auto-generated constraint names
PRIMARY KEY (id)  -- Will become user_movements_pkey (okay)
FOREIGN KEY (user_id) REFERENCES users(id)  -- Will get random name

-- ✅ GOOD: Explicit constraint names
CONSTRAINT pk_user_movements PRIMARY KEY (id),
CONSTRAINT fk_user_movements_user FOREIGN KEY (user_id) REFERENCES users(id)
```

## D. Schema Design Anti-Patterns (WARNING)

### Missing Audit Fields
```sql
-- ❌ BAD: No audit trail
CREATE TABLE movements (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL
);

-- ✅ GOOD: Full audit fields
CREATE TABLE movements (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
```

### Soft Delete Pattern
```sql
-- ❌ BAD: Hard delete only
-- No deleted_at column for soft deletes

-- ✅ GOOD: Soft delete support
ALTER TABLE movements ADD COLUMN deleted_at TIMESTAMPTZ;
CREATE INDEX idx_movements_deleted_at ON movements(deleted_at) WHERE deleted_at IS NOT NULL;
```

### Trigger Safety
```sql
-- ❌ BAD: Trigger without OR REPLACE
CREATE TRIGGER update_updated_at BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ✅ GOOD: Idempotent trigger creation
CREATE OR REPLACE TRIGGER update_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

## E. Foreign Key Policies (CRITICAL)

### Missing ON DELETE/UPDATE
```sql
-- ❌ BAD: No cascade policy (orphaned records risk)
FOREIGN KEY (user_id) REFERENCES users(id);

-- ✅ GOOD: Explicit policy
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE;
```

### Circular Dependencies
```sql
-- ❌ BAD: Circular FK without deferrable
Table A references Table B, Table B references Table A
-- Will fail during INSERT

-- ✅ GOOD: Deferrable constraints
CONSTRAINT fk_a_b FOREIGN KEY (b_id) REFERENCES b(id) DEFERRABLE INITIALLY DEFERRED;
```

## F. Type Safety (WARNING)

### Enum Safety
```sql
-- ❌ BAD: Enum without existence check
CREATE TYPE user_role AS ENUM ('user', 'admin');
-- Fails if enum already exists

-- ✅ GOOD: Idempotent enum creation
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
        CREATE TYPE user_role AS ENUM ('user', 'admin');
    END IF;
END $$;
```

### CHECK Constraints
```sql
-- ❌ BAD: No validation on numeric ranges
verification_tier SMALLINT NOT NULL

-- ✅ GOOD: Range validation
verification_tier SMALLINT NOT NULL DEFAULT 0,
CONSTRAINT chk_verification_tier CHECK (verification_tier >= 0 AND verification_tier <= 5)
```

## G. Performance (WARNING)

### Text Fields Without Length Limits
```sql
-- ❌ BAD: Unlimited TEXT fields (can cause bloat)
description TEXT,
name TEXT

-- ✅ GOOD: Length constraints
description TEXT,
name TEXT,
CONSTRAINT chk_name_length CHECK (char_length(name) <= 255)
```

### Missing Partial Indexes
```sql
-- ❌ BAD: Full table index when only filtering active records
CREATE INDEX idx_user_movements_status ON user_movements(status);

-- ✅ GOOD: Partial index for common query pattern
CREATE INDEX idx_user_movements_active ON user_movements(user_id)
WHERE status = 'active';
```

### Unused Indexes
```sql
-- ⚠️ WARNING: Redundant indexes
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_created_at_desc ON users(created_at DESC);
-- Second index is redundant (PostgreSQL can scan B-tree in reverse)
```

## H. Migration Safety (CRITICAL)

### Transaction Safety
```sql
-- ❌ BAD: Multi-statement migration without transaction
ALTER TABLE users ADD COLUMN foo TEXT;
ALTER TABLE users ADD COLUMN bar TEXT;
-- If second fails, first is still applied

-- ✅ GOOD: Transactional migration
BEGIN;
ALTER TABLE users ADD COLUMN foo TEXT;
ALTER TABLE users ADD COLUMN bar TEXT;
COMMIT;
```

### Rollback Validation
```sql
-- ❌ BAD: Irreversible migration without warning
DROP TABLE users;
DROP TYPE user_role;

-- ✅ GOOD: Safe rollback in .down.sql
-- Down migration should mirror up migration exactly
```

### Data Migration Safety
```sql
-- ❌ BAD: Unbounded UPDATE on large table
UPDATE users SET status = 'active' WHERE status IS NULL;
-- Locks entire table

-- ✅ GOOD: Batched update
DO $$
DECLARE
    batch_size INTEGER := 10000;
BEGIN
    LOOP
        UPDATE users SET status = 'active'
        WHERE status IS NULL
        AND ctid IN (SELECT ctid FROM users WHERE status IS NULL LIMIT batch_size);
        EXIT WHEN NOT FOUND;
        PERFORM pg_sleep(0.1);  -- Reduce lock contention
    END LOOP;
END $$;
```

## Spelling Detection

I maintain a dictionary of common SQL/PostgreSQL terms and domain-specific vocabulary:

### Database Terms
- `VARCHAR`, `TIMESTAMPTZ`, `UUID`, `BOOLEAN`, `SMALLINT`, `INTEGER`, `BIGINT`
- `CONSTRAINT`, `FOREIGN`, `REFERENCE`, `INDEX`, `TRIGGER`, `FUNCTION`
- `CASCADE`, `RESTRICT`, `SET NULL`, `DEFERRABLE`, `IMMEDIATE`
- `ENUM`, `CHECK`, `DEFAULT`, `NOT NULL`, `PRIMARY KEY`, `UNIQUE`

### Domain Terms (Standfor.me)
- `user`, `movement`, `organization`, `category`, `badge`, `verification`
- `advocacy`, `supporter`, `profile`, `visibility`, `status`

### Common Misspellings I Detect
- `INTOTO` → `INTO`
- `USERR` → `USER`
- `MOVEMET` → `MOVEMENT`
- `ORGANISATION` → `ORGANIZATION` (enforce US spelling consistency)
- `CATEGORIE` → `CATEGORY`
- `VISIBLIITY` → `VISIBILITY`
- `SUPPPORT` → `SUPPORT`
- `VERIFCATION` → `VERIFICATION`

## Output Format

I provide structured audit reports:

```markdown
# SQL Migration Audit Report

**Analyzed:** 18 files (9 up migrations, 9 down migrations)
**Date:** 2026-04-30

---

## Summary

| Severity | Count |
|----------|-------|
| 🔴 CRITICAL | 0 |
| 🟡 WARNING | 5 |
| 🔵 INFO | 12 |

---

## Findings by File

### 000001_bootstrap.up.sql
✅ No issues found

### 000002_create_users.up.sql
🟡 WARNING: Missing partial index for common query pattern
   - Line 45: idx_users_status covers all statuses, but queries often filter by status='active'
   - Suggestion: CREATE INDEX idx_users_active ON users(id) WHERE status = 'active';

🔵 INFO: Consider adding password expiration field for security compliance
   - Line 12: password_hash TEXT
   - Suggestion: Add password_expires_at TIMESTAMPTZ

### 000009_create_user_movements.up.sql
🟡 WARNING: Potential N+1 query pattern
   - Line 58: idx_user_movements_profile_display indexes (user_id, display_order, is_pinned)
   - Common query: SELECT ... WHERE user_id = $1 AND status = 'active' ORDER BY is_pinned DESC, display_order
   - Suggestion: Reorder index to (user_id, is_pinned, display_order) for better sort performance

🔵 INFO: Consistent naming convention used
   - All identifiers use snake_case
   - All constraints have explicit names
   - All indexes follow idx_table_column pattern

---

## Recommendations

### High Priority
1. Add composite index on (user_id, status) for user_movements table
2. Consider partial indexes for frequently filtered boolean/status columns

### Medium Priority
1. Standardize all COMMENT statements to use sentence case
2. Add CHECK constraints for all TEXT length validations

### Low Priority
1. Consider adding gin index on TEXT fields for full-text search
2. Document trigger behavior in comments
```

## Migration Pair Validation

I validate that `.up.sql` and `.down.sql` migrations are properly paired:

```markdown
### Migration Pair: 000009_create_user_movements

**Up Migration:** 000009_create_user_movements.up.sql ✅
**Down Migration:** 000009_create_user_movements.down.sql ✅

#### Rollback Safety Check
✅ DROP TABLE matches CREATE TABLE
✅ DROP TYPE matches CREATE TYPE (with IF EXISTS guards)
✅ DROP INDEX matches CREATE INDEX
✅ DROP TRIGGER matches CREATE TRIGGER

#### ⚠️ Warning
- Down migration drops type without IF EXISTS guard
  Suggestion: DROP TYPE IF EXISTS advocacy_status;
```

## Integration Commands

```bash
# Run SQL check on all migrations
make sql-check

# Run SQL check on specific file
make sql-check-one FILE=000009_create_user_movements.up.sql

# Run SQL check with CI output format
make sql-check-ci

# Generate SQL audit report
make sql-audit-report
```

## Pre-commit Integration

I can be integrated into pre-commit hooks:

```bash
# .git/hooks/pre-commit
#!/bin/bash
FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.sql$')
if [ -n "$FILES" ]; then
    echo "Running SQL migration check..."
    for file in $FILES; do
        if ! make sql-check-one FILE=$(basename $file); then
            echo "❌ SQL check failed for $file"
            exit 1
        fi
    done
fi
```

## Rules I Enforce

### Must (CRITICAL)
- ✅ All migrations must be idempotent (safe to run multiple times)
- ✅ All foreign keys must have explicit ON DELETE/UPDATE policies
- ✅ All dynamic SQL must use parameterized queries (format() with %I, %L)
- ✅ Enum types must use IF NOT EXISTS guards
- ✅ Triggers must use OR REPLACE

### Should (WARNING)
- ✅ All foreign key columns should have indexes
- ✅ All TEXT fields should have length CHECK constraints
- ✅ Index names should follow idx_table_column convention
- ✅ Constraint names should be explicit (pk_, fk_, uq_, chk_ prefixes)
- ✅ Multi-statement migrations should be wrapped in transactions
- ✅ Queries filtering on multiple columns should have composite indexes

### Nice to Have (INFO)
- ✅ Tables should have created_at, updated_at audit fields
- ✅ Tables should consider soft delete (deleted_at)
- ✅ Comments should use consistent casing (sentence case)
- ✅ Consider partial indexes for frequently filtered boolean columns
- ✅ Consider gin indexes for TEXT fields that will be searched

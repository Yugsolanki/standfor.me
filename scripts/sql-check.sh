#!/bin/bash

# ============================================================================
# SQL Migration Checker
# ============================================================================
# Performs strict static analysis on PostgreSQL migration files
# Detects vulnerabilities, anti-patterns, N+1 queries, and naming issues
# ============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Migration directory
MIGRATIONS_DIR="${MIGRATIONS_DIR:-backend/migrations}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Counters
CRITICAL_COUNT=0
WARNING_COUNT=0
INFO_COUNT=0
FILES_CHECKED=0
UP_MIGRATIONS=0
DOWN_MIGRATIONS=0

# Flags
CI_MODE=false
REPORT_MODE=false
SINGLE_FILE=""

# ============================================================================
# Helper Functions
# ============================================================================

print_header() {
    if [ "$CI_MODE" = true ]; then
        echo "## $1"
    else
        echo -e "${BOLD}${CYAN}=== $1 ===${NC}"
    fi
}

print_critical() {
    if [ "$CI_MODE" = true ]; then
        echo "**🔴 CRITICAL**: $1"
    else
        echo -e "${RED}🔴 CRITICAL${NC}: $1"
    fi
    ((CRITICAL_COUNT++)) || true
}

print_warning() {
    if [ "$CI_MODE" = true ]; then
        echo "**🟡 WARNING**: $1"
    else
        echo -e "${YELLOW}🟡 WARNING${NC}: $1"
    fi
    ((WARNING_COUNT++)) || true
}

print_info() {
    if [ "$CI_MODE" = true ]; then
        echo "**🔵 INFO**: $1"
    else
        echo -e "${BLUE}🔵 INFO${NC}: $1"
    fi
    ((INFO_COUNT++)) || true
}

print_success() {
    if [ "$CI_MODE" = true ]; then
        echo "**✅ PASSED**: $1"
    else
        echo -e "${GREEN}✅${NC} $1"
    fi
}

# ============================================================================
# Check Functions
# ============================================================================

check_spelling() {
    local file="$1"
    local content
    content=$(cat "$file")
    
    local filename
    filename=$(basename "$file")
    
    # Common misspellings (only obvious ones)
    local misspellings="INTOTO USERR MOVEMET ORGANISATION VISIBLIITY SUPPPORT VERIFCATION DISCRIPTION TIMETSAMP UUUID BOOLEN FOREING REFERNCES PRIMAY UNQIUE DEAFAULT MIGRAITON TRIGER FUNCITON INDXE FORIGN PRIVILEDGES SUCESS DEFAUT"
    
    for word in $misspellings; do
        if echo "$content" | grep -qiw "$word"; then
            local line_num
            line_num=$(grep -niw "$word" "$file" | head -1 | cut -d: -f1)
            local correction
            case "$word" in
                "INTOTO") correction="INTO" ;;
                "USERR") correction="USER" ;;
                "MOVEMET") correction="MOVEMENT" ;;
                "ORGANISATION") correction="ORGANIZATION" ;;
                "VISIBLIITY") correction="VISIBILITY" ;;
                "SUPPPORT") correction="SUPPORT" ;;
                "VERIFCATION") correction="VERIFICATION" ;;
                "DISCRIPTION") correction="DESCRIPTION" ;;
                "TIMETSAMP") correction="TIMESTAMP" ;;
                "UUUID") correction="UUID" ;;
                "BOOLEN") correction="BOOLEAN" ;;
                "FOREING"|"FORIGN") correction="FOREIGN" ;;
                "REFERNCES") correction="REFERENCES" ;;
                "PRIMAY") correction="PRIMARY" ;;
                "UNQIUE") correction="UNIQUE" ;;
                "DEAFAULT"|"DEFAUT") correction="DEFAULT" ;;
                "MIGRAITON") correction="MIGRATION" ;;
                "TRIGER") correction="TRIGGER" ;;
                "FUNCITON") correction="FUNCTION" ;;
                "INDXE") correction="INDEX" ;;
                "PRIVILEDGES") correction="PRIVILEGES" ;;
                "SUCESS") correction="SUCCESS" ;;
                *) correction="check spelling" ;;
            esac
            print_warning "$filename:$line_num - Possible spelling mistake: '$word' (did you mean '$correction'?)"
        fi
    done
}

check_sql_injection() {
    local file="$1"
    local content
    content=$(cat "$file")
    local filename
    filename=$(basename "$file")
    
    # Check for dangerous EXECUTE patterns
    if echo "$content" | grep -qiE "EXECUTE.*\|\|"; then
        local line_num
        line_num=$(grep -niE "EXECUTE.*\|\|" "$file" | head -1 | cut -d: -f1 || true)
        if [ -n "$line_num" ]; then
            print_critical "$filename:$line_num - Potential SQL injection: String concatenation in EXECUTE without parameterization"
        fi
    fi
}

check_foreign_key_policies() {
    local file="$1"
    local content
    content=$(cat "$file")
    local filename
    filename=$(basename "$file")
    
    # Check for FK without ON DELETE/UPDATE
    local fk_lines
    fk_lines=$(echo "$content" | grep -niE "FOREIGN KEY.*REFERENCES" | grep -v "ON DELETE" | grep -v "ON UPDATE" || true)
    
    if [ -n "$fk_lines" ]; then
        while IFS= read -r line; do
            local line_num
            line_num=$(echo "$line" | cut -d: -f1)
            print_warning "$filename:$line_num - Foreign key without explicit ON DELETE/UPDATE policy"
        done <<< "$fk_lines"
    fi
}

check_missing_indexes() {
    local file="$1"
    local content
    content=$(cat "$file")
    local filename
    filename=$(basename "$file")
    
    # Extract FK columns
    local fk_columns
    fk_columns=$(echo "$content" | grep -oE "FOREIGN KEY \([a-z_]+\)" | grep -oE "[a-z_]+" || true)
    
    # Check if indexes exist for FK columns
    for col in $fk_columns; do
        if ! echo "$content" | grep -qiE "CREATE.*INDEX.*($col)"; then
            print_warning "$filename - Foreign key column '$col' may benefit from an index for JOIN performance"
        fi
    done
}

check_naming_conventions() {
    local file="$1"
    local content
    content=$(cat "$file")
    local filename
    filename=$(basename "$file")
    
    # Check for PascalCase in table names (should be snake_case)
    if echo "$content" | grep -qiE "CREATE TABLE [A-Z][a-zA-Z]*"; then
        local line_num
        line_num=$(grep -niE "CREATE TABLE [A-Z]" "$file" | head -1 | cut -d: -f1 || true)
        if [ -n "$line_num" ]; then
            print_warning "$filename:$line_num - Table name uses PascalCase, consider snake_case for consistency"
        fi
    fi
    
    # Check index naming convention
    local indexes_without_prefix
    indexes_without_prefix=$(echo "$content" | grep -niE "CREATE.*INDEX" | grep -vi "idx_" || true)
    
    if [ -n "$indexes_without_prefix" ]; then
        while IFS= read -r line; do
            local line_num
            line_num=$(echo "$line" | cut -d: -f1)
            print_info "$filename:$line_num - Index name doesn't follow 'idx_table_column' convention"
        done <<< "$indexes_without_prefix"
    fi
}

check_enum_safety() {
    local file="$1"
    local content
    content=$(cat "$file")
    local filename
    filename=$(basename "$file")
    
    # Check for CREATE TYPE without IF NOT EXISTS
    if echo "$content" | grep -qiE "CREATE TYPE [a-z_]+ AS ENUM" && ! echo "$content" | grep -qiE "IF NOT EXISTS"; then
        local line_num
        line_num=$(grep -niE "CREATE TYPE.*AS ENUM" "$file" | head -1 | cut -d: -f1 || true)
        if [ -n "$line_num" ]; then
            print_warning "$filename:$line_num - CREATE TYPE without IF NOT EXISTS (not idempotent)"
        fi
    fi
}

check_trigger_safety() {
    local file="$1"
    local content
    content=$(cat "$file")
    local filename
    filename=$(basename "$file")
    
    # Check for CREATE TRIGGER without OR REPLACE
    if echo "$content" | grep -qiE "CREATE TRIGGER" && ! echo "$content" | grep -qiE "OR REPLACE"; then
        local line_num
        line_num=$(grep -niE "CREATE TRIGGER" "$file" | head -1 | cut -d: -f1 || true)
        if [ -n "$line_num" ]; then
            print_warning "$filename:$line_num - CREATE TRIGGER without OR REPLACE (not idempotent)"
        fi
    fi
}

check_transaction_safety() {
    local file="$1"
    local content
    content=$(cat "$file")
    local filename
    filename=$(basename "$file")
    
    # Count ALTER/CREATE/DROP statements
    local statement_count
    statement_count=$(echo "$content" | grep -ciE "^(ALTER|CREATE|DROP)" 2>/dev/null | tr -d "\n" || true)
    statement_count=${statement_count:-0}
    if ! [[ "$statement_count" =~ ^[0-9]+$ ]]; then statement_count=0; fi
    
    if [ "$statement_count" -gt 1 ] && ! echo "$content" | grep -qiE "BEGIN"; then
        print_warning "$filename - Multi-statement migration without explicit transaction (BEGIN/COMMIT)"
    fi
}

check_audit_fields() {
    local file="$1"
    local content
    content=$(cat "$file")
    local filename
    filename=$(basename "$file")
    
    # Check for CREATE TABLE without audit fields
    if echo "$content" | grep -qiE "CREATE TABLE"; then
        if ! echo "$content" | grep -qiE "created_at.*TIMESTAMPTZ|created_at.*TIMESTAMP"; then
            print_info "$filename - Consider adding created_at TIMESTAMPTZ for audit trail"
        fi
        
        if ! echo "$content" | grep -qiE "updated_at.*TIMESTAMPTZ|updated_at.*TIMESTAMP"; then
            print_info "$filename - Consider adding updated_at TIMESTAMPTZ with trigger for automatic updates"
        fi
        
        if ! echo "$content" | grep -qiE "deleted_at.*TIMESTAMPTZ|deleted_at.*TIMESTAMP"; then
            print_info "$filename - Consider adding deleted_at for soft delete support"
        fi
    fi
}

check_text_constraints() {
    local file="$1"
    local content
    content=$(cat "$file")
    local filename
    filename=$(basename "$file")
    
    # Check for TEXT columns without length constraints (skip common long fields)
    local text_columns
    text_columns=$(echo "$content" | grep -E "[[:space:]][a-z_]+ TEXT" | grep -v "CONSTRAINT" | grep -v "CHECK" || true)
    
    if [ -n "$text_columns" ]; then
        while IFS= read -r line; do
            local col_name
            col_name=$(echo "$line" | grep -oE "[a-z_]+ TEXT" | awk '{print $1}')
            # Skip if it's a common exception
            if [[ ! "$col_name" =~ ^(long_description|description|bio)$ ]]; then
                print_info "$filename - TEXT column '$col_name' without length CHECK constraint"
            fi
        done <<< "$text_columns"
    fi
}

check_n_plus_one_patterns() {
    local file="$1"
    local content
    content=$(cat "$file")
    local filename
    filename=$(basename "$file")
    
    # Look for single-column indexes that could be part of composite indexes
    local single_column_indexes
    single_column_indexes=$(echo "$content" | grep -niE "CREATE INDEX.*ON [a-z_]+\\([a-z_]+\\)$" || true)
    
    if [ -n "$single_column_indexes" ]; then
        while IFS= read -r line; do
            local line_num
            line_num=$(echo "$line" | cut -d: -f1)
            
            # Extract column from index definition
            if [[ "$line" =~ ON[[:space:]]+([a-z_]+)\(([a-z_]+)\)$ ]]; then
                local col_name="${BASH_REMATCH[2]}"
                
                # Check if there's a WHERE clause filtering on other columns
                if echo "$content" | grep -qiE "WHERE.*AND.*$col_name" || \
                   echo "$content" | grep -qiE "WHERE.*$col_name.*AND"; then
                    print_warning "$filename:$line_num - Single-column index on '$col_name' may cause N+1 queries when filtering with other columns"
                fi
            fi
        done <<< "$single_column_indexes"
    fi
}

check_migration_pair() {
    local up_file="$1"
    local filename
    filename=$(basename "$up_file" .up.sql)
    local down_file="${MIGRATIONS_DIR}/${filename}.down.sql"
    
    if [ ! -f "$down_file" ]; then
        print_critical "$filename - Missing corresponding .down.sql migration file"
        return
    fi
    
    local up_content down_content
    up_content=$(cat "$up_file")
    down_content=$(cat "$down_file")
    
    # Check if CREATE TABLE has corresponding DROP TABLE
    if echo "$up_content" | grep -qiE "CREATE TABLE"; then
        if ! echo "$down_content" | grep -qiE "DROP TABLE"; then
            print_warning "$filename - CREATE TABLE without DROP TABLE in down migration"
        fi
    fi
    
    # Check if down migration has IF EXISTS guards
    if echo "$down_content" | grep -qiE "DROP (TABLE|INDEX|TYPE)" && \
       ! echo "$down_content" | grep -qiE "DROP.*IF EXISTS"; then
        print_info "$filename - Down migration DROP statements without IF EXISTS guards"
    fi
}

# ============================================================================
# Main Analysis Function
# ============================================================================

analyze_file() {
    local file="$1"
    local filename
    filename=$(basename "$file")
    
    ((FILES_CHECKED++)) || true
    
    if [[ "$file" == *".up.sql" ]]; then
        ((UP_MIGRATIONS++)) || true
    elif [[ "$file" == *".down.sql" ]]; then
        ((DOWN_MIGRATIONS++)) || true
    fi
    
    if [ "$CI_MODE" = false ] && [ "$REPORT_MODE" = false ]; then
        echo -e "\n${BOLD}Analyzing: $filename${NC}"
    fi
    
    # Run all checks
    check_spelling "$file"
    check_sql_injection "$file"
    check_foreign_key_policies "$file"
    check_missing_indexes "$file"
    check_naming_conventions "$file"
    check_enum_safety "$file"
    check_trigger_safety "$file"
    check_transaction_safety "$file"
    check_audit_fields "$file"
    check_text_constraints "$file"
    check_n_plus_one_patterns "$file"
    
    # Check migration pairs (only for up migrations)
    if [[ "$file" == *".up.sql" ]]; then
        check_migration_pair "$file"
    fi
}

# ============================================================================
# Report Generation
# ============================================================================

generate_report() {
    local report_file="sql-migration-audit-report.md"
    
    cat > "$report_file" << EOF
# SQL Migration Audit Report

**Generated:** $(date '+%Y-%m-%d %H:%M:%S')
**Directory:** $MIGRATIONS_DIR
**Files Analyzed:** $FILES_CHECKED

---

## Summary

| Severity | Count |
|----------|-------|
| 🔴 CRITICAL | $CRITICAL_COUNT |
| 🟡 WARNING | $WARNING_COUNT |
| 🔵 INFO | $INFO_COUNT |

---

## Analysis Details

Detailed findings have been printed above.

---

## Recommendations

### High Priority
$(if [ $CRITICAL_COUNT -gt 0 ]; then echo "- Address all CRITICAL issues before deployment"; else echo "- ✅ No critical issues found"; fi)

### Medium Priority
$(if [ $WARNING_COUNT -gt 0 ]; then echo "- Review and fix WARNING-level issues"; else echo "- ✅ No warnings found"; fi)

### Low Priority
- Consider implementing INFO-level suggestions for best practices

---

## Migration Coverage

- **Up Migrations:** $UP_MIGRATIONS
- **Down Migrations:** $DOWN_MIGRATIONS
- **Files Checked:** $FILES_CHECKED

---

*Generated by SQL Migration Checker*
EOF
    
    echo -e "\n${GREEN}✓ Report generated: $report_file${NC}"
}

print_summary() {
    echo ""
    print_header "Summary"
    echo ""
    echo "Files Analyzed:    $FILES_CHECKED ($UP_MIGRATIONS up, $DOWN_MIGRATIONS down)"
    
    if [ "$CI_MODE" = true ]; then
        echo ""
        echo "**🔴 CRITICAL:** $CRITICAL_COUNT"
        echo "**🟡 WARNING:** $WARNING_COUNT"
        echo "**🔵 INFO:** $INFO_COUNT"
        echo ""
        
        if [ $CRITICAL_COUNT -gt 0 ]; then
            echo "## Result: ❌ FAILED"
            exit 1
        elif [ $WARNING_COUNT -gt 0 ]; then
            echo "## Result: ⚠️ PASSED WITH WARNINGS"
        else
            echo "## Result: ✅ PASSED"
        fi
    else
        echo ""
        echo -e "🔴 CRITICAL: ${RED}$CRITICAL_COUNT${NC}"
        echo -e "🟡 WARNING:  ${YELLOW}$WARNING_COUNT${NC}"
        echo -e "🔵 INFO:     ${BLUE}$INFO_COUNT${NC}"
        echo ""
        
        if [ $CRITICAL_COUNT -gt 0 ]; then
            echo -e "${RED}❌ SQL check FAILED with $CRITICAL_COUNT critical issue(s)${NC}"
            exit 1
        elif [ $WARNING_COUNT -gt 0 ]; then
            echo -e "${YELLOW}⚠️  SQL check PASSED with $WARNING_COUNT warning(s)${NC}"
        else
            echo -e "${GREEN}✅ SQL check PASSED${NC}"
        fi
    fi
    echo ""
}

# ============================================================================
# Main
# ============================================================================

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --file)
                SINGLE_FILE="$2"
                shift 2
                ;;
            --ci)
                CI_MODE=true
                shift
                ;;
            --report)
                REPORT_MODE=true
                shift
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --file FILE   Check a single migration file"
                echo "  --ci          CI-friendly output format"
                echo "  --report      Generate detailed report"
                echo "  --help        Show this help message"
                echo ""
                echo "Environment:"
                echo "  MIGRATIONS_DIR  Path to migrations directory (default: backend/migrations)"
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    cd "$PROJECT_ROOT"
    
    if [ "$CI_MODE" = false ]; then
        print_header "SQL Migration Checker"
        echo ""
    fi
    
    # Find files to check
    local files=()
    
    if [ -n "$SINGLE_FILE" ]; then
        if [ -f "$MIGRATIONS_DIR/$SINGLE_FILE" ]; then
            files=("$MIGRATIONS_DIR/$SINGLE_FILE")
        else
            echo "Error: File not found: $MIGRATIONS_DIR/$SINGLE_FILE"
            exit 1
        fi
    else
        # Find all .sql files in migrations directory
        while IFS= read -r -d '' file; do
            files+=("$file")
        done < <(find "$MIGRATIONS_DIR" -type f -name "*.sql" -print0 | sort -z)
    fi
    
    if [ ${#files[@]} -eq 0 ]; then
        echo "No SQL files found in $MIGRATIONS_DIR"
        exit 0
    fi
    
    # Analyze each file
    for file in "${files[@]}"; do
        analyze_file "$file"
    done
    
    # Print summary and generate report if requested
    if [ "$REPORT_MODE" = true ]; then
        generate_report
    fi
    
    print_summary
}

main "$@"

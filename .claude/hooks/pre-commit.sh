#!/bin/bash
# Pre-Commit Hook - Runs before git commit
# Enforces linting and code formatting compliance
#
# This hook runs as a Claude PreToolUse hook before git commit commands.
# It checks staged files for linting errors and formatting issues.
#
# Exit codes:
# - 0: All checks passed, proceed with commit
# - 1: Warnings (commit proceeds but user notified)
# - 2: Blocked (commit aborted)

# Read tool input from stdin (when run as Claude hook)
INPUT=$(cat 2>/dev/null || echo "{}")

echo "🔍 Running pre-commit checks (linting & formatting)..."
echo ""

EXIT_CODE=0
LINT_ERRORS=""
FORMAT_ERRORS=""

# Get staged files
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM 2>/dev/null)

if [ -z "$STAGED_FILES" ]; then
    echo "  ℹ️  No staged files to check"
    exit 0
fi

# ============================================
# BRANCH CHECK
# ============================================

BRANCH=$(git branch --show-current 2>/dev/null)
if [[ "$BRANCH" == "main" || "$BRANCH" == "master" ]]; then
    if [ "${ALLOW_MAIN_COMMIT:-0}" != "1" ]; then
        echo "  ⚠️  Warning: Committing directly to $BRANCH"
        echo "     Consider using a feature branch instead."
        echo ""
    fi
fi

# ============================================
# LINTING CHECKS
# ============================================

echo "📋 Running linters..."

# JavaScript/TypeScript (ESLint)
JS_FILES=$(echo "$STAGED_FILES" | grep -E '\.(js|jsx|ts|tsx)$' || true)
if [ -n "$JS_FILES" ]; then
    if command -v npx &> /dev/null && [ -f "package.json" ]; then
        # Try ESLint
        if [ -f ".eslintrc.js" ] || [ -f ".eslintrc.json" ] || [ -f ".eslintrc.yml" ] || [ -f "eslint.config.js" ] || grep -q "eslint" package.json 2>/dev/null; then
            echo "  → ESLint: Checking JS/TS files..."
            ESLINT_OUTPUT=$(echo "$JS_FILES" | xargs npx eslint --no-error-on-unmatched-pattern 2>&1) || {
                LINT_ERRORS="${LINT_ERRORS}ESLint errors:\n${ESLINT_OUTPUT}\n\n"
                EXIT_CODE=2
            }
        fi
    fi
fi

# Python (Ruff or Flake8)
PY_FILES=$(echo "$STAGED_FILES" | grep -E '\.py$' || true)
if [ -n "$PY_FILES" ]; then
    if command -v ruff &> /dev/null; then
        echo "  → Ruff: Checking Python files..."
        RUFF_OUTPUT=$(echo "$PY_FILES" | xargs ruff check 2>&1) || {
            LINT_ERRORS="${LINT_ERRORS}Ruff errors:\n${RUFF_OUTPUT}\n\n"
            EXIT_CODE=2
        }
    elif command -v flake8 &> /dev/null; then
        echo "  → Flake8: Checking Python files..."
        FLAKE8_OUTPUT=$(echo "$PY_FILES" | xargs flake8 2>&1) || {
            LINT_ERRORS="${LINT_ERRORS}Flake8 errors:\n${FLAKE8_OUTPUT}\n\n"
            EXIT_CODE=2
        }
    fi
fi

# Go (golint/staticcheck)
GO_FILES=$(echo "$STAGED_FILES" | grep -E '\.go$' || true)
if [ -n "$GO_FILES" ]; then
    if command -v staticcheck &> /dev/null; then
        echo "  → Staticcheck: Checking Go files..."
        STATICCHECK_OUTPUT=$(echo "$GO_FILES" | xargs staticcheck 2>&1) || {
            LINT_ERRORS="${LINT_ERRORS}Staticcheck errors:\n${STATICCHECK_OUTPUT}\n\n"
            EXIT_CODE=2
        }
    elif command -v golint &> /dev/null; then
        echo "  → Golint: Checking Go files..."
        GOLINT_OUTPUT=$(echo "$GO_FILES" | xargs golint 2>&1)
        if [ -n "$GOLINT_OUTPUT" ]; then
            LINT_ERRORS="${LINT_ERRORS}Golint warnings:\n${GOLINT_OUTPUT}\n\n"
            # golint is advisory, don't block
        fi
    fi
fi

# Rust (clippy)
RS_FILES=$(echo "$STAGED_FILES" | grep -E '\.rs$' || true)
if [ -n "$RS_FILES" ]; then
    if command -v cargo &> /dev/null && [ -f "Cargo.toml" ]; then
        echo "  → Clippy: Checking Rust files..."
        CLIPPY_OUTPUT=$(cargo clippy --message-format=short 2>&1) || {
            LINT_ERRORS="${LINT_ERRORS}Clippy errors:\n${CLIPPY_OUTPUT}\n\n"
            EXIT_CODE=2
        }
    fi
fi

# Shell scripts (shellcheck)
SH_FILES=$(echo "$STAGED_FILES" | grep -E '\.(sh|bash)$' || true)
if [ -n "$SH_FILES" ]; then
    if command -v shellcheck &> /dev/null; then
        echo "  → ShellCheck: Checking shell scripts..."
        SHELLCHECK_OUTPUT=$(echo "$SH_FILES" | xargs shellcheck 2>&1) || {
            LINT_ERRORS="${LINT_ERRORS}ShellCheck errors:\n${SHELLCHECK_OUTPUT}\n\n"
            EXIT_CODE=2
        }
    fi
fi

# ============================================
# FORMATTING COMPLIANCE CHECKS
# ============================================

echo ""
echo "🎨 Checking code formatting..."

# JavaScript/TypeScript/Web (Prettier)
WEB_FILES=$(echo "$STAGED_FILES" | grep -E '\.(js|jsx|ts|tsx|json|md|css|html|vue|svelte)$' || true)
if [ -n "$WEB_FILES" ]; then
    if command -v npx &> /dev/null && [ -f "package.json" ]; then
        if [ -f ".prettierrc" ] || [ -f ".prettierrc.json" ] || [ -f ".prettierrc.js" ] || [ -f "prettier.config.js" ] || grep -q "prettier" package.json 2>/dev/null; then
            echo "  → Prettier: Checking formatting..."
            PRETTIER_OUTPUT=$(echo "$WEB_FILES" | xargs npx prettier --check 2>&1) || {
                FORMAT_ERRORS="${FORMAT_ERRORS}Prettier formatting issues:\n${PRETTIER_OUTPUT}\n\n"
                EXIT_CODE=2
            }
        fi
    fi
fi

# Python (Black)
if [ -n "$PY_FILES" ]; then
    if command -v black &> /dev/null; then
        echo "  → Black: Checking Python formatting..."
        BLACK_OUTPUT=$(echo "$PY_FILES" | xargs black --check --quiet 2>&1) || {
            FORMAT_ERRORS="${FORMAT_ERRORS}Black formatting issues (run 'black <file>' to fix):\n$(echo "$PY_FILES" | tr '\n' ' ')\n\n"
            EXIT_CODE=2
        }
    fi
fi

# Go (gofmt)
if [ -n "$GO_FILES" ]; then
    if command -v gofmt &> /dev/null; then
        echo "  → gofmt: Checking Go formatting..."
        GOFMT_OUTPUT=$(echo "$GO_FILES" | xargs gofmt -l 2>&1)
        if [ -n "$GOFMT_OUTPUT" ]; then
            FORMAT_ERRORS="${FORMAT_ERRORS}gofmt formatting issues (run 'gofmt -w <file>' to fix):\n${GOFMT_OUTPUT}\n\n"
            EXIT_CODE=2
        fi
    fi
fi

# Rust (rustfmt)
if [ -n "$RS_FILES" ]; then
    if command -v rustfmt &> /dev/null; then
        echo "  → rustfmt: Checking Rust formatting..."
        RUSTFMT_OUTPUT=$(echo "$RS_FILES" | xargs rustfmt --check 2>&1) || {
            FORMAT_ERRORS="${FORMAT_ERRORS}rustfmt formatting issues (run 'rustfmt <file>' to fix):\n$(echo "$RS_FILES" | tr '\n' ' ')\n\n"
            EXIT_CODE=2
        }
    fi
fi

# Shell scripts (shfmt)
if [ -n "$SH_FILES" ]; then
    if command -v shfmt &> /dev/null; then
        echo "  → shfmt: Checking shell script formatting..."
        SHFMT_OUTPUT=$(echo "$SH_FILES" | xargs shfmt -d 2>&1)
        if [ -n "$SHFMT_OUTPUT" ]; then
            FORMAT_ERRORS="${FORMAT_ERRORS}shfmt formatting issues (run 'shfmt -w <file>' to fix):\n$(echo "$SH_FILES" | tr '\n' ' ')\n\n"
            EXIT_CODE=2
        fi
    fi
fi

# ============================================
# YAML SYNTAX VALIDATION
# ============================================

echo ""
echo "📄 Validating YAML syntax..."

YAML_FILES=$(echo "$STAGED_FILES" | grep -E '\.(yml|yaml)$' || true)
if [ -n "$YAML_FILES" ]; then
    # Check if PyYAML is available
    if ! python3 -c "import yaml" 2>/dev/null; then
        echo "  ⚠️  PyYAML not installed - YAML validation skipped"
        echo "     Install with: pip install pyyaml"
    else
        # Batch all YAML files into single Python invocation for performance
        # Pass filenames via stdin to avoid command-line length limits and injection
        YAML_OUTPUT=$(echo "$YAML_FILES" | python3 << 'PYEOF'
import sys
import yaml

errors = []
for line in sys.stdin:
    filepath = line.strip()
    if not filepath:
        continue
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            yaml.safe_load(f)
    except yaml.YAMLError as e:
        errors.append(f"{filepath}: {e}")
    except Exception as e:
        errors.append(f"{filepath}: {e}")

if errors:
    for err in errors:
        print(err)
    sys.exit(1)
sys.exit(0)
PYEOF
)
        YAML_RESULT=$?

        if [ $YAML_RESULT -ne 0 ]; then
            echo "  ⛔ YAML syntax errors detected:"
            echo "$YAML_OUTPUT" | sed 's/^/     /'
            EXIT_CODE=2
        else
            echo "  ✓ All YAML files valid"
        fi
    fi
else
    echo "  ℹ️  No YAML files in commit"
fi

# ============================================
# SECURITY CHECKS
# ============================================

echo ""
echo "🔒 Running security checks..."

# Filter files for security scanning - exclude directories that contain legitimate patterns:
# - .github/ contains workflow files with GITHUB_TOKEN env var references
# - .claude/ contains hooks/docs that describe security patterns to detect
# - docs/ may contain security documentation with example patterns
SECURITY_FILES=$(echo "$STAGED_FILES" | grep -vE '^\.github/|^\.claude/|^docs/' | grep -vE '\.(md|yml|yaml)$' || true)

# Look for sensitive data patterns (secrets/credentials)
# Only scan actual code files, not infrastructure/documentation
if [ -n "$SECURITY_FILES" ]; then
    # Support both quoted (with spaces) and unquoted values:
    # - Quoted: API_KEY="my secret passphrase" or API_KEY='secret'
    # - Unquoted: API_KEY=mysecret123
    # Uses alternation: ("..."|'...'|unquoted) to capture full quoted strings
    SENSITIVE_PATTERNS="API_KEY=(\"[^\"]+\"|'[^']+'|[^'\"\s]+)|SECRET=(\"[^\"]+\"|'[^']+'|[^'\"\s]+)|PASSWORD=(\"[^\"]+\"|'[^']+'|[^'\"\s]+)|-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY"
    SENSITIVE_FOUND=$(echo "$SECURITY_FILES" | xargs grep -l -E "$SENSITIVE_PATTERNS" 2>/dev/null | grep -v ".example" | grep -v ".template" | head -5)
    if [ -n "$SENSITIVE_FOUND" ]; then
        echo "  ⛔ Potential secrets found in:"
        echo "$SENSITIVE_FOUND" | sed 's/^/     /'
        echo "     Please review before committing."
        EXIT_CODE=2
    fi
fi

# Verify no .env files are being committed
ENV_FILES=$(echo "$STAGED_FILES" | grep -E '^\.env$|\.env\.local$|\.env\.production$')
if [ -n "$ENV_FILES" ]; then
    echo "  ⛔ Environment files staged for commit:"
    echo "$ENV_FILES" | sed 's/^/     /'
    echo "     These should be in .gitignore."
    EXIT_CODE=2
fi

# Check for debugging artifacts
DEBUG_PATTERNS="console\.log|debugger|print\(.*#.*debug|binding\.pry|import pdb"
DEBUG_FOUND=$(echo "$STAGED_FILES" | xargs grep -l -E "$DEBUG_PATTERNS" 2>/dev/null | head -5)
if [ -n "$DEBUG_FOUND" ]; then
    echo "  ⚠️  Debug statements found in:"
    echo "$DEBUG_FOUND" | sed 's/^/     /'
    echo "     Consider removing before commit."
fi

# ============================================
# PII (PERSONAL INFORMATION) SCAN
# ============================================

echo ""
echo "👤 Scanning for personal information (PII)..."

PII_ERRORS=""

# Skip binary files, config files, and infrastructure files that may contain false positives
# (workflow files, hook files, documentation, config files with IDs)
CODE_FILES=$(echo "$STAGED_FILES" | grep -vE '\.(png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot|pdf|zip|tar|gz|md|toml|yml|yaml)$' | grep -vE '(\.github/|\.claude/)' || true)

# IMPORTANT: All PII patterns BLOCK commits because this is a public repo.
# Once committed, data is permanently in git history and exposed.

# Helper function to report PII with file:line (content masked for security)
# Uses batch processing for performance with safe filename handling
# Usage: report_pii_matches <pattern> <excludes> <label>
report_pii_matches() {
    local pattern="$1"
    local excludes="$2"
    local label="$3"
    local result=""

    # Build file array for safe handling of filenames with spaces
    local -a file_array=()
    while IFS= read -r file; do
        [ -z "$file" ] && continue
        [ -f "$file" ] && file_array+=("$file")
    done <<< "$(echo "$CODE_FILES" | tr ' ' '\n')"

    # Return early if no files to scan (prevents grep from hanging on STDIN)
    [ ${#file_array[@]} -eq 0 ] && return

    # Batch grep with proper quoting for filenames with spaces
    local matches
    if [ -n "$excludes" ]; then
        matches=$(grep -nHE "$pattern" "${file_array[@]}" 2>/dev/null | grep -vE "$excludes" | head -10)
    else
        matches=$(grep -nHE "$pattern" "${file_array[@]}" 2>/dev/null | head -10)
    fi

    if [ -n "$matches" ]; then
        while IFS= read -r match; do
            # Format: file:line:content
            local file_path="${match%%:*}"
            local rest="${match#*:}"
            local line_num="${rest%%:*}"
            local content="${rest#*:}"

            # Mask content: show first 3 chars, then *** (protects PII in CI logs)
            local masked_content
            if [ ${#content} -gt 3 ]; then
                masked_content="${content:0:3}***"
            else
                masked_content="***"
            fi

            result="${result}     ${file_path}:${line_num}: ${masked_content}\n"
        done <<< "$matches"
    fi

    echo -e "$result"
}

if [ -n "$CODE_FILES" ]; then
    # Email addresses (but exclude example.com, test.com, localhost patterns)
    EMAIL_PATTERN='[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}'
    EMAIL_EXCLUDES='example\.com|test\.com|localhost|your-?email|user@|email@|foo@|bar@|noreply@|no-reply@|users\.noreply\.github\.com'
    EMAIL_MATCHES=$(report_pii_matches "$EMAIL_PATTERN" "$EMAIL_EXCLUDES" "Email")
    if [ -n "$EMAIL_MATCHES" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ Email addresses found:\n${EMAIL_MATCHES}"
        EXIT_CODE=2
    fi

    # Phone numbers (various formats: +1-xxx-xxx-xxxx, (xxx) xxx-xxxx, xxx.xxx.xxxx)
    PHONE_PATTERN='\+?1?[-.\s]?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}'
    PHONE_MATCHES=$(report_pii_matches "$PHONE_PATTERN" "" "Phone")
    if [ -n "$PHONE_MATCHES" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ Phone numbers found:\n${PHONE_MATCHES}"
        EXIT_CODE=2
    fi

    # Social Security Numbers (xxx-xx-xxxx format)
    SSN_PATTERN='[0-9]{3}-[0-9]{2}-[0-9]{4}'
    SSN_MATCHES=$(report_pii_matches "$SSN_PATTERN" "" "SSN")
    if [ -n "$SSN_MATCHES" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ SSN patterns found:\n${SSN_MATCHES}"
        EXIT_CODE=2
    fi

    # Credit card numbers (basic patterns for major card types)
    CC_PATTERN='[3-6][0-9]{3}[-\s]?[0-9]{4}[-\s]?[0-9]{4}[-\s]?[0-9]{4}'
    CC_MATCHES=$(report_pii_matches "$CC_PATTERN" "" "Credit Card")
    if [ -n "$CC_MATCHES" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ Credit card patterns found:\n${CC_MATCHES}"
        EXIT_CODE=2
    fi

    # IP addresses (but exclude common private/localhost ranges)
    IP_PATTERN='[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}'
    IP_EXCLUDES='127\.0\.0\.1|0\.0\.0\.0|192\.168\.|10\.|172\.(1[6-9]|2[0-9]|3[0-1])\.|localhost'
    IP_MATCHES=$(report_pii_matches "$IP_PATTERN" "$IP_EXCLUDES" "IP Address")
    if [ -n "$IP_MATCHES" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ Public IP addresses found:\n${IP_MATCHES}"
        EXIT_CODE=2
    fi

    # AWS Account IDs (12 digits in aws/arn context)
    AWS_PATTERN='(aws|arn:|account).{0,20}[0-9]{12}'
    AWS_MATCHES=$(report_pii_matches "$AWS_PATTERN" "" "AWS Account ID")
    if [ -n "$AWS_MATCHES" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ AWS Account ID patterns found:\n${AWS_MATCHES}"
        EXIT_CODE=2
    fi

    # Physical addresses (basic pattern: number + street name)
    ADDR_PATTERN='[0-9]+\s+(N\.?|S\.?|E\.?|W\.?|North|South|East|West)?\s*[A-Z][a-z]+\s+(St\.?|Street|Ave\.?|Avenue|Rd\.?|Road|Blvd\.?|Boulevard|Dr\.?|Drive|Ln\.?|Lane|Way|Ct\.?|Court)'
    ADDR_EXCLUDES='test|mock|example|sample'
    ADDR_MATCHES=$(report_pii_matches "$ADDR_PATTERN" "$ADDR_EXCLUDES" "Address")
    if [ -n "$ADDR_MATCHES" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ Physical addresses found:\n${ADDR_MATCHES}"
        EXIT_CODE=2
    fi

    # Full names (First Last pattern in specific contexts)
    NAME_PATTERN='(name|author|user|contact|owner|created[_ ]?by|assigned[_ ]?to|submitted[_ ]?by)\s*[:=]?\s*[A-Z][a-z]+\s+[A-Z][a-z]+'
    NAME_EXCLUDES='Hello World|Lorem Ipsum|Foo Bar|John Doe|Jane Doe|Test User|Example User|First Last|Your Name'
    NAME_MATCHES=$(report_pii_matches "$NAME_PATTERN" "$NAME_EXCLUDES" "Full Name")
    if [ -n "$NAME_MATCHES" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ Full names found:\n${NAME_MATCHES}"
        EXIT_CODE=2
    fi
fi

if [ -n "$PII_ERRORS" ]; then
    echo -e "$PII_ERRORS"
    echo ""
    echo "     ⛔ COMMIT BLOCKED: Personal information detected!"
    echo "     This is a PUBLIC repository - data in git history is permanent."
    echo "     Remove PII and use placeholders (e.g., user@example.com)."
fi

# ============================================
# REPORT RESULTS
# ============================================

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ -n "$LINT_ERRORS" ]; then
    echo ""
    echo "⛔ LINTING ERRORS:"
    echo "-----------------"
    echo -e "$LINT_ERRORS"
fi

if [ -n "$FORMAT_ERRORS" ]; then
    echo ""
    echo "⛔ FORMATTING ISSUES:"
    echo "--------------------"
    echo -e "$FORMAT_ERRORS"
    echo ""
    echo "💡 Tip: Run the formatter on these files before committing."
    echo "   The PostToolUse hook auto-formats on Write/Edit, but manual"
    echo "   changes may need formatting."
fi

echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ Pre-commit checks passed - all linting and formatting OK"
elif [ $EXIT_CODE -eq 1 ]; then
    echo "⚠️  Pre-commit checks passed with warnings"
else
    echo "⛔ Pre-commit checks FAILED - commit blocked"
    echo ""
    echo "   Fix the issues above before committing."
    echo "   To bypass (not recommended): git commit --no-verify"
fi

exit $EXIT_CODE

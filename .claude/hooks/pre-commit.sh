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

# Look for sensitive data patterns (secrets/credentials)
SENSITIVE_PATTERNS="API_KEY=|SECRET=|PASSWORD=|PRIVATE_KEY|-----BEGIN"
SENSITIVE_FOUND=$(echo "$STAGED_FILES" | xargs grep -l -E "$SENSITIVE_PATTERNS" 2>/dev/null | grep -v ".example" | grep -v ".template" | head -5)
if [ -n "$SENSITIVE_FOUND" ]; then
    echo "  ⛔ Potential secrets found in:"
    echo "$SENSITIVE_FOUND" | sed 's/^/     /'
    echo "     Please review before committing."
    EXIT_CODE=2
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

if [ -n "$CODE_FILES" ]; then
    # Email addresses (but exclude example.com, test.com, localhost patterns)
    EMAIL_PATTERN='[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}'
    EMAIL_EXCLUDES='example\.com|test\.com|localhost|your-?email|user@|email@|foo@|bar@|noreply@|no-reply@|users\.noreply\.github\.com'
    EMAIL_FOUND=$(echo "$CODE_FILES" | xargs grep -lE "$EMAIL_PATTERN" 2>/dev/null | while read -r file; do
        if grep -E "$EMAIL_PATTERN" "$file" 2>/dev/null | grep -vE "$EMAIL_EXCLUDES" | grep -qE "$EMAIL_PATTERN"; then
            echo "$file"
        fi
    done | head -5)
    if [ -n "$EMAIL_FOUND" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ Email addresses found in:\n"
        PII_ERRORS="${PII_ERRORS}$(echo "$EMAIL_FOUND" | sed 's/^/     /')\n"
        EXIT_CODE=2
    fi

    # Phone numbers (various formats: +1-xxx-xxx-xxxx, (xxx) xxx-xxxx, xxx.xxx.xxxx)
    PHONE_PATTERN='\+?1?[-.\s]?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}'
    PHONE_FOUND=$(echo "$CODE_FILES" | xargs grep -lE "$PHONE_PATTERN" 2>/dev/null | head -3)
    if [ -n "$PHONE_FOUND" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ Phone numbers found in:\n"
        PII_ERRORS="${PII_ERRORS}$(echo "$PHONE_FOUND" | sed 's/^/     /')\n"
        EXIT_CODE=2
    fi

    # Social Security Numbers (xxx-xx-xxxx format)
    SSN_PATTERN='[0-9]{3}-[0-9]{2}-[0-9]{4}'
    SSN_FOUND=$(echo "$CODE_FILES" | xargs grep -lE "$SSN_PATTERN" 2>/dev/null | head -3)
    if [ -n "$SSN_FOUND" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ SSN patterns found in:\n"
        PII_ERRORS="${PII_ERRORS}$(echo "$SSN_FOUND" | sed 's/^/     /')\n"
        EXIT_CODE=2
    fi

    # Credit card numbers (basic patterns for major card types)
    # Visa: 4xxx, Mastercard: 5xxx, Amex: 3xxx, etc.
    CC_PATTERN='[3-6][0-9]{3}[-\s]?[0-9]{4}[-\s]?[0-9]{4}[-\s]?[0-9]{4}'
    CC_FOUND=$(echo "$CODE_FILES" | xargs grep -lE "$CC_PATTERN" 2>/dev/null | head -3)
    if [ -n "$CC_FOUND" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ Credit card patterns found in:\n"
        PII_ERRORS="${PII_ERRORS}$(echo "$CC_FOUND" | sed 's/^/     /')\n"
        EXIT_CODE=2
    fi

    # IP addresses (but exclude common private/localhost ranges in certain contexts)
    IP_PATTERN='[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}'
    IP_EXCLUDES='127\.0\.0\.1|0\.0\.0\.0|192\.168\.|10\.|172\.(1[6-9]|2[0-9]|3[0-1])\.|localhost'
    IP_FOUND=$(echo "$CODE_FILES" | xargs grep -lE "$IP_PATTERN" 2>/dev/null | while read -r file; do
        if grep -E "$IP_PATTERN" "$file" 2>/dev/null | grep -vE "$IP_EXCLUDES" | grep -qE "$IP_PATTERN"; then
            echo "$file"
        fi
    done | head -3)
    if [ -n "$IP_FOUND" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ Public IP addresses found in:\n"
        PII_ERRORS="${PII_ERRORS}$(echo "$IP_FOUND" | sed 's/^/     /')\n"
        EXIT_CODE=2
    fi

    # AWS Account IDs (12 digits)
    AWS_PATTERN='[0-9]{12}'
    # Only check in specific contexts to reduce false positives
    AWS_FOUND=$(echo "$CODE_FILES" | xargs grep -lE "(aws|arn:|account).{0,20}$AWS_PATTERN" 2>/dev/null | head -3)
    if [ -n "$AWS_FOUND" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ AWS Account ID patterns found in:\n"
        PII_ERRORS="${PII_ERRORS}$(echo "$AWS_FOUND" | sed 's/^/     /')\n"
        EXIT_CODE=2
    fi

    # Physical addresses (basic pattern: number + street name)
    ADDR_PATTERN='[0-9]+\s+(N\.?|S\.?|E\.?|W\.?|North|South|East|West)?\s*[A-Z][a-z]+\s+(St\.?|Street|Ave\.?|Avenue|Rd\.?|Road|Blvd\.?|Boulevard|Dr\.?|Drive|Ln\.?|Lane|Way|Ct\.?|Court)'
    ADDR_FOUND=$(echo "$CODE_FILES" | xargs grep -lE "$ADDR_PATTERN" 2>/dev/null | grep -v "test" | grep -v "mock" | grep -v "example" | head -3)
    if [ -n "$ADDR_FOUND" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ Physical addresses found in:\n"
        PII_ERRORS="${PII_ERRORS}$(echo "$ADDR_FOUND" | sed 's/^/     /')\n"
        EXIT_CODE=2
    fi

    # Full names (First Last pattern - capitalized words that look like names)
    # Look for patterns like "name: John Smith" or "author: Jane Doe" or "by John Smith"
    NAME_CONTEXT='(name|author|user|contact|owner|created[_ ]?by|assigned[_ ]?to|submitted[_ ]?by)\s*[:=]?\s*'
    NAME_PATTERN="[A-Z][a-z]+\s+[A-Z][a-z]+"
    NAME_EXCLUDES='Hello World|Lorem Ipsum|Foo Bar|John Doe|Jane Doe|Test User|Example User|First Last|Your Name'
    NAME_FOUND=$(echo "$CODE_FILES" | xargs grep -lE "${NAME_CONTEXT}${NAME_PATTERN}" 2>/dev/null | while read -r file; do
        if grep -E "${NAME_CONTEXT}${NAME_PATTERN}" "$file" 2>/dev/null | grep -vE "$NAME_EXCLUDES" | grep -qE "${NAME_CONTEXT}${NAME_PATTERN}"; then
            echo "$file"
        fi
    done | head -3)
    if [ -n "$NAME_FOUND" ]; then
        PII_ERRORS="${PII_ERRORS}  ⛔ Full names found in:\n"
        PII_ERRORS="${PII_ERRORS}$(echo "$NAME_FOUND" | sed 's/^/     /')\n"
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

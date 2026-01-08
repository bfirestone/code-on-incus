#!/bin/bash
# Documentation coverage checker (similar to yard-lint for Ruby)
# Checks that all exported functions, types, and constants have godoc comments

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Checking documentation coverage..."
echo ""

# Find all Go files except tests and generated files
GO_FILES=$(find . -type f -name "*.go" \
  -not -path "*/vendor/*" \
  -not -path "*_test.go" \
  -not -name "*_generated.go")

MISSING_DOCS=()
TOTAL_EXPORTED=0
DOCUMENTED=0

# Check each file
for file in $GO_FILES; do
  # Extract exported symbols (functions, types, constants, vars)
  # Format: "Type:Name" (e.g., "func:MyFunction", "type:MyStruct")
  SYMBOLS=$(awk '
    # Track if we just saw a comment
    /^\/\// { has_comment = 1; next }
    /^\/\*/ { in_comment = 1; next }
    /\*\/$/ { in_comment = 0; has_comment = 1; next }

    # Skip blank lines between comment and declaration
    /^$/ && has_comment { next }

    # Match exported declarations
    /^(func|type|const|var) [A-Z]/ {
      if (!in_comment) {
        # Extract type and name
        match($0, /^(func|type|const|var) ([A-Z][a-zA-Z0-9_]*)/, arr)
        if (arr[2]) {
          symbol = arr[1] ":" arr[2]
          if (has_comment) {
            print "DOCUMENTED:" symbol
          } else {
            print "MISSING:" symbol
          }
        }
        has_comment = 0
      }
    }

    # Reset comment flag for non-matching lines
    !/^\/\// && !/^\/\*/ && !/\*\/$/ && !/^$/ { has_comment = 0; in_comment = 0 }
  ' "$file" | while read line; do
    status="${line%%:*}"
    symbol="${line#*:}"

    if [ -n "$symbol" ]; then
      TOTAL_EXPORTED=$((TOTAL_EXPORTED + 1))

      if [ "$status" = "DOCUMENTED" ]; then
        DOCUMENTED=$((DOCUMENTED + 1))
      else
        # Store for summary
        echo "$file:$symbol" >> /tmp/missing_docs.txt
      fi
    fi
  done
done

# Calculate coverage from temp file
if [ -f /tmp/missing_docs.txt ]; then
  MISSING_COUNT=$(wc -l < /tmp/missing_docs.txt)
else
  MISSING_COUNT=0
fi

# Re-count total (awkward but works with subshells)
TOTAL_EXPORTED=$(find . -type f -name "*.go" \
  -not -path "*/vendor/*" \
  -not -path "*_test.go" \
  -not -name "*_generated.go" \
  -exec grep -E "^(func|type|const|var) [A-Z]" {} \; | wc -l)

DOCUMENTED=$((TOTAL_EXPORTED - MISSING_COUNT))

# Calculate percentage
if [ "$TOTAL_EXPORTED" -gt 0 ]; then
  COVERAGE=$(awk "BEGIN {printf \"%.1f\", ($DOCUMENTED / $TOTAL_EXPORTED) * 100}")
else
  COVERAGE=100
fi

# Display results
echo "======================================"
echo "  Documentation Coverage Report"
echo "======================================"
echo ""
echo "Total exported symbols:  $TOTAL_EXPORTED"
echo "Documented:              $DOCUMENTED"
echo "Missing documentation:   $MISSING_COUNT"
echo ""

if [ "$MISSING_COUNT" -eq 0 ]; then
  echo -e "${GREEN}✓ Documentation coverage: ${COVERAGE}%${NC}"
  echo ""
  echo "All exported symbols are documented! 🎉"
  rm -f /tmp/missing_docs.txt
  exit 0
else
  echo -e "${YELLOW}⚠ Documentation coverage: ${COVERAGE}%${NC}"
  echo ""
  echo "Missing documentation for:"
  echo ""

  while IFS= read -r line; do
    file="${line%%:*}"
    symbol="${line#*:}"
    type="${symbol%%:*}"
    name="${symbol#*:}"
    echo -e "  ${RED}✗${NC} $file"
    echo "    ${type}: $name"
  done < /tmp/missing_docs.txt

  echo ""
  echo -e "${YELLOW}Add godoc comments for all exported symbols${NC}"
  echo "Format: // Name does something..."

  rm -f /tmp/missing_docs.txt

  # Fail if coverage is below threshold (80%)
  THRESHOLD=80
  if awk "BEGIN {exit !($COVERAGE < $THRESHOLD)}"; then
    echo ""
    echo -e "${RED}✗ Documentation coverage ${COVERAGE}% is below threshold ${THRESHOLD}%${NC}"
    exit 1
  fi

  exit 0
fi

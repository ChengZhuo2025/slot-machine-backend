#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# 调整后的覆盖率阈值
OVERALL_MIN="${OVERALL_MIN:-70}"
KEY_MODULE_MIN="${KEY_MODULE_MIN:-85}"

# 测试产物输出目录
OUTPUT_DIR="${OUTPUT_DIR:-$ROOT_DIR/tests/output}"
mkdir -p "$OUTPUT_DIR"

# Coverage profile path (must match scripts/coverage.sh default unless overridden)
COVERPROFILE="${COVERPROFILE:-$OUTPUT_DIR/coverage.out}"

calc_stats() {
	local patterns="${1:-}" # semicolon-separated substrings; empty means all
	awk -v pats="$patterns" '
BEGIN {
	n = (pats == "" ? 0 : split(pats, p, ";"))
}
NR == 1 { next } # mode line
{
	split($1, a, ":")
	path = a[1]

	if (pats != "") {
		ok = 0
		for (i = 1; i <= n; i++) {
			if (p[i] != "" && index(path, p[i]) > 0) { ok = 1; break }
		}
		if (!ok) next
	}

	# With -coverpkg and multi-package go test runs, coverprofiles may contain duplicate blocks.
	# Deduplicate by block range ($1) + statement count ($2) and aggregate counts across duplicates.
	key = $1 " " $2
	if (!(key in stmt_map)) {
		stmt_map[key] = ($2 + 0)
	}
	cnt_map[key] += ($3 + 0)
}
END {
	for (k in stmt_map) {
		total += stmt_map[k]
		if (cnt_map[k] > 0) covered += stmt_map[k]
	}
	if (total == 0) {
		printf "0.00 0 0\n"
		exit 0
	}
	pct = (covered * 100.0) / total
	printf "%.2f %d %d\n", pct, total, covered
}
' "$COVERPROFILE"
}

pct_ge() {
	awk -v a="$1" -v b="$2" 'BEGIN { exit !((a + 0) >= (b + 0)) }'
}

echo "Running coverage gate..."
echo " - overall >= ${OVERALL_MIN}%"
echo " - key modules >= ${KEY_MODULE_MIN}% (auth/payment/order/rental/booking)"
echo " - output directory: ${OUTPUT_DIR}"
echo

# Generate coverage profile (skip HTML by default for speed).
# Default includes API/E2E tests to measure real business flows; override via GO_TEST_TAGS/GO_TEST_TARGETS/COVERPKG.
GO_TEST_TAGS="${GO_TEST_TAGS:-api,e2e}"
GO_TEST_TARGETS="${GO_TEST_TARGETS:-./...}"
COVERPKG="${COVERPKG:-./internal/service/auth/...,./internal/service/payment/...,./internal/service/order/...,./internal/service/rental/...,./internal/service/hotel/...}"
GENERATE_HTML="${GENERATE_HTML:-0}" \
	OUTPUT_DIR="$OUTPUT_DIR" \
	COVERPROFILE="$COVERPROFILE" \
	GO_TEST_TAGS="$GO_TEST_TAGS" \
	GO_TEST_TARGETS="$GO_TEST_TARGETS" \
	COVERPKG="$COVERPKG" \
	bash "$ROOT_DIR/scripts/coverage.sh"

echo
overall_line="$(calc_stats "")"
overall_pct="$(echo "$overall_line" | awk '{print $1}')"
overall_total="$(echo "$overall_line" | awk '{print $2}')"
overall_covered="$(echo "$overall_line" | awk '{print $3}')"
printf "Overall coverage: %s%% (%s/%s statements)\n" "$overall_pct" "$overall_covered" "$overall_total"

module_patterns() {
	case "$1" in
	auth) echo "internal/service/auth/" ;;
	payment) echo "internal/service/payment/" ;;
	order) echo "internal/service/order/" ;;
	rental) echo "internal/service/rental/" ;;
	booking) echo "internal/service/hotel/booking_;internal/service/hotel/code_service.go" ;;
	*) echo "" ;;
	esac
}

failed=0

for module in auth payment order rental booking; do
	pats="$(module_patterns "$module")"
	line="$(calc_stats "$pats")"
	pct="$(echo "$line" | awk '{print $1}')"
	total="$(echo "$line" | awk '{print $2}')"
	covered="$(echo "$line" | awk '{print $3}')"

	if [[ "$total" == "0" ]]; then
		printf "%-8s: no statements matched patterns: %s\n" "$module" "$pats"
		failed=1
		continue
	fi

	printf "%-8s: %s%% (%s/%s statements)\n" "$module" "$pct" "$covered" "$total"
	if ! pct_ge "$pct" "$KEY_MODULE_MIN"; then
		failed=1
	fi
done

echo

if ! pct_ge "$overall_pct" "$OVERALL_MIN"; then
	failed=1
fi

if [[ "$failed" -ne 0 ]]; then
	echo "Coverage gate: FAIL" >&2
	exit 1
fi

echo "Coverage gate: PASS"
echo "Coverage artifacts saved to: ${OUTPUT_DIR}"

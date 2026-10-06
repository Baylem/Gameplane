#!/bin/sh
# check-telemetry-default.sh: fail unless the project default telemetry
# destination is set to an https URL (spec 022, research R13).
#
# api/internal/telemetry/destination.go holds DefaultEndpoint, the one place
# the project URL lives. It stays empty until OPEN-DECISIONS OD-1 is ruled, so
# this check is EXPECTED to fail until then. It is the merge gate for the
# feature PR and runs as its own CI job (telemetry-default-gate); it is
# deliberately not part of `make lint`.
#
# Exit codes:
#   0 = DefaultEndpoint is non-empty and starts with https://
#   1 = it is empty, not https, or the constant could not be found

set -eu

cd "$(dirname "$0")/.."
file="${TELEMETRY_DESTINATION_FILE:-api/internal/telemetry/destination.go}"

if [ ! -r "$file" ]; then
	echo "check-telemetry-default: cannot read $file" >&2
	exit 1
fi

line=$(grep -E '^const DefaultEndpoint = "' "$file" || true)
if [ -z "$line" ]; then
	echo "check-telemetry-default: no 'const DefaultEndpoint = \"...\"' line in $file" >&2
	exit 1
fi

value=$(printf '%s\n' "$line" | sed -e 's/^const DefaultEndpoint = "//' -e 's/".*$//')
case "$value" in
https://?*)
	echo "check-telemetry-default: DefaultEndpoint is set ($value)"
	;;
"")
	echo "check-telemetry-default: DefaultEndpoint is empty; rule OPEN-DECISIONS OD-1 and set it before merging" >&2
	exit 1
	;;
*)
	echo "check-telemetry-default: DefaultEndpoint must start with https:// (got: $value)" >&2
	exit 1
	;;
esac

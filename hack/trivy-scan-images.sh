#!/usr/bin/env bash
# trivy-scan-images.sh: Spec 013 G2 (contracts/static-gates.md). Scans each
# image argument with Trivy and fails if any image has a HIGH or CRITICAL
# vulnerability that has a fix. Every image is scanned before the exit status
# is decided, so one run lists every finding.
#
# Fix findings with a base-image digest bump or a Go/npm dependency bump in
# the image's module, never an ignore file (FR-008).
#
# Usage: hack/trivy-scan-images.sh IMAGE [IMAGE...]
# Env:   TRIVY - Trivy binary to run (default: trivy)
#
# Exit codes: 0 = every image clean; 1 = at least one image failed;
#             2 = no image given
set -euo pipefail

if [[ $# -eq 0 ]]; then
  echo "usage: $0 IMAGE [IMAGE...]" >&2
  exit 2
fi

trivy_bin="${TRIVY:-trivy}"
failed=()

for img in "$@"; do
  echo "::group::trivy ${img}"
  if "$trivy_bin" image --severity HIGH,CRITICAL --ignore-unfixed --exit-code 1 --no-progress "$img"; then
    echo "✓ ${img}"
  else
    failed+=("$img")
  fi
  echo "::endgroup::"
done

if [[ ${#failed[@]} -gt 0 ]]; then
  printf '✗ fixed HIGH/CRITICAL vulnerabilities (or a scan error) in %d of %d image(s):\n' "${#failed[@]}" "$#" >&2
  printf '  %s\n' "${failed[@]}" >&2
  exit 1
fi
echo "✓ $# image(s) clean"

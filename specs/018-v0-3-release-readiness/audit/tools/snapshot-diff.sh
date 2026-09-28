#!/usr/bin/env bash
# Purpose: Compare two cluster snapshots and report changes
# Usage: snapshot-diff.sh <before-dir> <after-dir>
# Exit codes: 0 = no mismatches, 1 = mismatches found, 2 = usage error, missing files, or incomplete snapshots
set -euo pipefail

: "${KUBECONFIG:=$HOME/kubelab.yaml}"; export KUBECONFIG

# Check for jq
if ! command -v jq &>/dev/null; then
  echo "ERROR: jq is required but not installed" >&2
  exit 2
fi

# Validate usage
if [[ $# -ne 2 ]]; then
  echo "Usage: snapshot-diff.sh <before-dir> <after-dir>" >&2
  exit 2
fi

BEFORE_DIR="$1"
AFTER_DIR="$2"

# Check that directories exist
if [[ ! -d "$BEFORE_DIR" ]] || [[ ! -d "$AFTER_DIR" ]]; then
  echo "ERROR: One or both directories do not exist" >&2
  exit 2
fi

mismatch_found=0

# Files to compare (excluding images.json, helm-list.json, helm-values.json)
declare -a CRD_KINDS=("gameservers" "gametemplates" "backups" "backupschedules" "restores" "modules" "modulesources" "networkcaptures" "clusters")

compare_crd() {
  local kind="$1"
  local filename="crd-${kind}.json"
  local before_file="$BEFORE_DIR/$filename"
  local after_file="$AFTER_DIR/$filename"

  if [[ ! -f "$before_file" ]]; then
    echo "ERROR: $filename not found in before-dir (incomplete baseline)" >&2
    exit 2
  fi

  if [[ ! -f "$after_file" ]]; then
    echo "ERROR: $filename not found in after-dir" >&2
    mismatch_found=1
    return
  fi

  # Create keyed versions and compare, excluding audit018- objects
  local before_keyed=$(jq -r '.[] | select(.name | startswith("audit018-") | not) | "\(.kind)/\(.namespace)/\(.name) uid=\(.uid) gen=\(.generation)"' "$before_file" | sort)
  local after_keyed=$(jq -r '.[] | select(.name | startswith("audit018-") | not) | "\(.kind)/\(.namespace)/\(.name) uid=\(.uid) gen=\(.generation)"' "$after_file" | sort)

  # Check for MISSING
  while IFS= read -r line; do
    if [[ -z "$line" ]]; then continue; fi
    local key=$(echo "$line" | cut -d' ' -f1)
    if ! grep -q "^$key " <<< "$after_keyed"; then
      echo "MISSING: $line"
      mismatch_found=1
    fi
  done <<< "$before_keyed"

  # Check for UID/generation mismatches
  while IFS= read -r before_line; do
    if [[ -z "$before_line" ]]; then continue; fi
    local key=$(echo "$before_line" | cut -d' ' -f1)
    local after_line=$(grep "^$key " <<< "$after_keyed" || true)

    if [[ -n "$after_line" ]]; then
      local before_uid=$(echo "$before_line" | grep -oP 'uid=\K[^ ]+')
      local after_uid=$(echo "$after_line" | grep -oP 'uid=\K[^ ]+')

      if [[ "$before_uid" != "$after_uid" ]]; then
        echo "UID MISMATCH: $key ($before_uid → $after_uid)"
        mismatch_found=1
      fi

      local before_gen=$(echo "$before_line" | grep -oP 'gen=\K[^ ]+')
      local after_gen=$(echo "$after_line" | grep -oP 'gen=\K[^ ]+')

      if [[ "$before_gen" != "$after_gen" ]]; then
        echo "GENERATION MISMATCH: $key ($before_gen → $after_gen)"
        mismatch_found=1
      fi
    fi
  done <<< "$before_keyed"

  # Check for NEW (warning)
  while IFS= read -r line; do
    if [[ -z "$line" ]]; then continue; fi
    local key=$(echo "$line" | cut -d' ' -f1)
    if ! grep -q "^$key " <<< "$before_keyed"; then
      echo "NEW (warning): $line"
      mismatch_found=1
    fi
  done <<< "$after_keyed"
}

# Compare CRD files
for kind in "${CRD_KINDS[@]}"; do
  compare_crd "$kind"
done

# Compare PVCs
if [[ -f "$BEFORE_DIR/pvcs.json" ]] && [[ -f "$AFTER_DIR/pvcs.json" ]]; then
  pvcs_before=$(jq -r '.[] | select(.name | startswith("audit018-") | not) | "\(.namespace)/\(.name) uid=\(.uid) volumeName=\(.volumeName)"' "$BEFORE_DIR/pvcs.json" | sort)
  pvcs_after=$(jq -r '.[] | select(.name | startswith("audit018-") | not) | "\(.namespace)/\(.name) uid=\(.uid) volumeName=\(.volumeName)"' "$AFTER_DIR/pvcs.json" | sort)

  # Check for MISSING
  while IFS= read -r line; do
    if [[ -z "$line" ]]; then continue; fi
    key=$(echo "$line" | cut -d' ' -f1)
    if ! grep -q "^$key " <<< "$pvcs_after"; then
      echo "MISSING PVC: $line"
      mismatch_found=1
    fi
  done <<< "$pvcs_before"

  # Check for UID mismatches
  while IFS= read -r before_line; do
    if [[ -z "$before_line" ]]; then continue; fi
    key=$(echo "$before_line" | cut -d' ' -f1)
    after_line=$(grep "^$key " <<< "$pvcs_after" || true)

    if [[ -n "$after_line" ]]; then
      before_uid=$(echo "$before_line" | grep -oP 'uid=\K[^ ]+')
      after_uid=$(echo "$after_line" | grep -oP 'uid=\K[^ ]+')

      if [[ "$before_uid" != "$after_uid" ]]; then
        echo "PVC UID MISMATCH: $key ($before_uid → $after_uid)"
        mismatch_found=1
      fi
    fi
  done <<< "$pvcs_before"

  # Check for NEW
  while IFS= read -r line; do
    if [[ -z "$line" ]]; then continue; fi
    key=$(echo "$line" | cut -d' ' -f1)
    if ! grep -q "^$key " <<< "$pvcs_before"; then
      echo "NEW PVC (warning): $line"
      mismatch_found=1
    fi
  done <<< "$pvcs_after"
fi

# Compare nodes
if [[ -f "$BEFORE_DIR/nodes.json" ]] && [[ -f "$AFTER_DIR/nodes.json" ]]; then
  nodes_before=$(jq -r '.[] | "\(.name) uid=\(.uid) schedulable=\(.schedulable) roles=\(.roles | join(",")) kubeletVersion=\(.kubeletVersion)"' "$BEFORE_DIR/nodes.json" | sort)
  nodes_after=$(jq -r '.[] | "\(.name) uid=\(.uid) schedulable=\(.schedulable) roles=\(.roles | join(",")) kubeletVersion=\(.kubeletVersion)"' "$AFTER_DIR/nodes.json" | sort)

  # Check for MISSING nodes
  while IFS= read -r line; do
    if [[ -z "$line" ]]; then continue; fi
    name=$(echo "$line" | cut -d' ' -f1)
    if ! grep -q "^$name " <<< "$nodes_after"; then
      echo "MISSING NODE: $line"
      mismatch_found=1
    fi
  done <<< "$nodes_before"

  # Check for UID mismatches
  while IFS= read -r before_line; do
    if [[ -z "$before_line" ]]; then continue; fi
    name=$(echo "$before_line" | cut -d' ' -f1)
    after_line=$(grep "^$name " <<< "$nodes_after" || true)

    if [[ -n "$after_line" ]]; then
      before_uid=$(echo "$before_line" | grep -oP 'uid=\K[^ ]+')
      after_uid=$(echo "$after_line" | grep -oP 'uid=\K[^ ]+')

      if [[ "$before_uid" != "$after_uid" ]]; then
        echo "NODE UID MISMATCH: $name ($before_uid → $after_uid)"
        mismatch_found=1
      fi
    fi
  done <<< "$nodes_before"

  # Check for NEW nodes
  while IFS= read -r line; do
    if [[ -z "$line" ]]; then continue; fi
    name=$(echo "$line" | cut -d' ' -f1)
    if ! grep -q "^$name " <<< "$nodes_before"; then
      echo "NEW NODE (warning): $line"
      mismatch_found=1
    fi
  done <<< "$nodes_after"
fi

# Skip images.json
echo "NOTE: images.json skipped (Gameplane Deployments/StatefulSets/DaemonSets change on purpose during RC upgrades)"

# Skip helm-list.json
echo "NOTE: helm-list.json skipped (Gameplane release changes on purpose)"

# Compare helm-values.json with allowlist of permitted changes
if [[ -f "$BEFORE_DIR/helm-values.json" ]] && [[ -f "$AFTER_DIR/helm-values.json" ]]; then
  # Allowlist of keys that are permitted to change during procedures
  # These are the only Helm values that procedures are documented to modify
  declare -a ALLOWED_HELM_CHANGES=(
    "ingress"
    "networkPolicies"
    "agent"
    "api"
  )

  # Create jq filter to extract only allowlisted keys
  filter="{$(for key in "${ALLOWED_HELM_CHANGES[@]}"; do echo "\"$key\": .$key"; done | paste -sd, -)}"

  # Extract only allowlisted paths and compare
  helm_before=$(jq "$filter" "$BEFORE_DIR/helm-values.json" 2>/dev/null || echo "{}")
  helm_after=$(jq "$filter" "$AFTER_DIR/helm-values.json" 2>/dev/null || echo "{}")

  # Simple diff: if the allowlisted values differ, flag it
  if [[ "$helm_before" != "$helm_after" ]]; then
    # Check if any NON-allowlisted keys differ
    all_before=$(jq '.' "$BEFORE_DIR/helm-values.json")
    all_after=$(jq '.' "$AFTER_DIR/helm-values.json")

    # Extract keys that are NOT in the allowlist
    other_keys=$(jq -r 'keys | .[]' <<< "$all_before" | while read key; do
      skip=0
      for allowed in "${ALLOWED_HELM_CHANGES[@]}"; do
        if [[ "$key" == "$allowed" ]]; then
          skip=1
          break
        fi
      done
      if [[ $skip -eq 0 ]]; then
        echo "$key"
      fi
    done)

    # Check if non-allowlisted keys changed
    unexpected_change=0
    while IFS= read -r key; do
      [[ -z "$key" ]] && continue
      before_val=$(jq ".\"$key\"" <<< "$all_before" 2>/dev/null)
      after_val=$(jq ".\"$key\"" <<< "$all_after" 2>/dev/null)
      if [[ "$before_val" != "$after_val" ]]; then
        echo "HELM VALUE MISMATCH: $key changed unexpectedly"
        mismatch_found=1
        unexpected_change=1
      fi
    done <<< "$other_keys"

    if [[ $unexpected_change -eq 0 ]]; then
      # Only allowlisted keys changed, which is acceptable
      echo "NOTE: helm-values.json changed only in permitted keys (ingress, networkPolicies, agent, api)"
    fi
  fi
else
  echo "NOTE: helm-values.json skipped (one or both files missing)"
fi

if [[ $mismatch_found -eq 1 ]]; then
  exit 1
fi

exit 0

# Review verification index (T045)

One link per tier-up verification file under `evidence/review-*/`. Each file lists every candidate from that component's `notes.md` with a kept/rejected verdict, a severity and a reason, and gives the repro for each kept candidate. Kept candidates are recorded in [findings.md](../findings.md) with origin `review:<component>`. Rejected candidates stay only in the verification file, with their reason. Security-sensitive candidates are verified separately, off-git, and are not linked from here ([OD-019](../../OPEN-DECISIONS.md#od-019-security-findings-stay-unpushed-until-fixed--resolved-2026-09-24)).

| Component (coverage row) | Verification |
|---|---|
| agent/ | [evidence/review-agent/verification.md](review-agent/verification.md) |
| api/ | [evidence/review-api/verification.md](review-api/verification.md) |
| audit-syslog-bridge/ | [evidence/review-audit-syslog-bridge/verification.md](review-audit-syslog-bridge/verification.md) |
| capture-sidecar/ | [evidence/review-capture-sidecar/verification.md](review-capture-sidecar/verification.md) |
| charts/gameplane/ | [evidence/review-charts-gameplane/verification.md](review-charts-gameplane/verification.md) |
| deploy/ | [evidence/review-deploy/verification.md](review-deploy/verification.md) |
| docs/, modules/, website/ | [evidence/review-docs/verification.md](review-docs/verification.md) |
| cross-component follow-up items (F-251 to F-255) | [evidence/review-followup/verification.md](review-followup/verification.md) |
| gameaction/ | [evidence/review-gameaction/verification.md](review-gameaction/verification.md) |
| gameproto/ | [evidence/review-gameproto/verification.md](review-gameproto/verification.md) |
| .github/actions/ | [evidence/review-github-actions/verification.md](review-github-actions/verification.md) |
| .github/workflows/ | [evidence/review-github-workflows/verification.md](review-github-workflows/verification.md) |
| gp-module/ | [evidence/review-gp-module/verification.md](review-gp-module/verification.md) |
| hack/ | [evidence/review-hack/verification.md](review-hack/verification.md) |
| images/ | [evidence/review-images/verification.md](review-images/verification.md) |
| mcp-server/ | [evidence/review-mcp-server/verification.md](review-mcp-server/verification.md) |
| netguard/ | [evidence/review-netguard/verification.md](review-netguard/verification.md) |
| operator/ | [evidence/review-operator/verification.md](review-operator/verification.md) |
| sentinel/ | [evidence/review-sentinel/verification.md](review-sentinel/verification.md) |
| svcutil/ | [evidence/review-svcutil/verification.md](review-svcutil/verification.md) |
| telemetry-receiver/ | [evidence/review-telemetry-receiver/verification.md](review-telemetry-receiver/verification.md) |
| test/e2e/ | [evidence/review-test-e2e/verification.md](review-test-e2e/verification.md) |
| tunnel/ | [evidence/review-tunnel/verification.md](review-tunnel/verification.md) |
| web/ | [evidence/review-web/verification.md](review-web/verification.md) |

The `root docs` and `design-export/` rows in [coverage.md](../coverage.md) have no verification file of their own. Their findings (F-030, F-043) were filed with origin `review:root-docs`, design-export recorded none, and F-253 (root `README.md`) came through the follow-up chunk above.

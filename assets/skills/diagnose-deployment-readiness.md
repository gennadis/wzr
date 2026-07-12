# Skill: diagnose-deployment-readiness

Analyze database health metrics and Jenkins build status to produce a deployment readiness verdict.

## Parameters

- `service` — service name (e.g. `payments-api`)
- `environment` — target environment (e.g. `staging`, `production`)

## Instructions

1. Parse the context from the previous steps:
   - Database health metrics: `error_rate_pct`, `p95_latency_ms`, `failed_records_24h`, `total_records`
   - Jenkins build: result (`SUCCESS` / `FAILURE`), build number, test pass rate
2. Apply these thresholds:
   - `error_rate_pct` < 1%
   - `p95_latency_ms` < 500
   - `failed_records_24h` = 0
   - Jenkins result = `SUCCESS`
   - Test pass rate ≥ 98%
3. For each threshold record PASS or FAIL and the actual value.
4. Output a verdict:
   - **DEPLOY SAFE** — all thresholds met.
   - **DEPLOY BLOCKED** — one or more thresholds failed. List each failing metric with its actual value and the threshold it violated.
5. For DEPLOY BLOCKED, state the minimum actions required before deployment can proceed.

## Expected output

```
Deployment readiness: {{ .service }} → {{ .environment }}

Metrics:
  Error rate:       0.3%   PASS  (threshold < 1%)
  p95 latency:      220ms  PASS  (threshold < 500ms)
  Failed records:   0      PASS
  Jenkins build:    #142 SUCCESS PASS
  Test pass rate:   99.1%  PASS  (threshold ≥ 98%)

Verdict: DEPLOY SAFE

--- or ---

Verdict: DEPLOY BLOCKED
Failing checks:
  - p95 latency: 820ms  FAIL  (threshold < 500ms)
  - Failed records: 14   FAIL  (must be 0)

Required actions before deploying:
  - Investigate latency spike in payments-api — check slow query log
  - Resolve 14 failed records in the last 24h before promoting to production
```

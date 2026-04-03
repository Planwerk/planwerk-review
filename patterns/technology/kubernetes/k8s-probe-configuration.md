# Review Pattern: Kubernetes Probe Configuration

**Review-Area**: quality
**Detection-Hint**: Pods without livenessProbe or readinessProbe, probes pointing to the same endpoint, aggressive probe timings
**Severity**: WARNING
**Category**: technology
**Applies-When**: kubernetes, helm
**Sources**: Configure Liveness, Readiness and Startup Probes (https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/), Production Best Practices (https://learnk8s.io/production-best-practices)

## What to check

1. Every long-running container should have a `readinessProbe` (controls traffic routing)
2. `livenessProbe` should check a different condition than `readinessProbe` — liveness checks if the process is stuck, readiness checks if it can serve traffic
3. `startupProbe` should be used for slow-starting applications instead of high `initialDelaySeconds`
4. Probe `timeoutSeconds` and `periodSeconds` should be reasonable (not 1s for a database health check)
5. Liveness probes must NOT check downstream dependencies — a database outage should not restart your pods
6. `failureThreshold * periodSeconds` determines how long before action — ensure it matches your SLO

## Why it matters

Misconfigured probes are the #1 cause of unnecessary pod restarts and cascading failures. A liveness probe that checks a downstream dependency can turn a partial outage into a complete cluster meltdown.

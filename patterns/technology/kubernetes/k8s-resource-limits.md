# Review Pattern: Kubernetes Resource Limits

**Review-Area**: quality
**Detection-Hint**: Pod specs or container specs without resources.requests or resources.limits, missing CPU/memory constraints
**Severity**: CRITICAL
**Category**: technology
**Applies-When**: kubernetes, helm
**Sources**: Kubernetes Best Practices: Resource Requests and Limits (https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/), Production Best Practices (https://learnk8s.io/production-best-practices)

## What to check

1. Every container must have `resources.requests` for both CPU and memory
2. Every container must have `resources.limits` for memory (CPU limits are debatable but memory limits are mandatory)
3. Requests should not exceed limits
4. Limits should be reasonable — not `999Gi` or `1000` CPU
5. Init containers also need resource specifications

## Why it matters

Without resource limits, a single pod can starve the entire node. Without requests, the scheduler cannot make informed placement decisions, leading to overcommitted nodes and OOM kills.

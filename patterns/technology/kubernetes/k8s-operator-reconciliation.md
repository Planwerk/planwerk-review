# Review Pattern: Kubernetes Operator Reconciliation

**Review-Area**: quality
**Detection-Hint**: Reconcile functions with side effects on repeated calls, missing status updates, non-idempotent resource creation, missing error handling in reconcile loops
**Severity**: CRITICAL
**Category**: technology
**Applies-When**: kubernetes
**Sources**: Operator SDK Common Recommendations (https://sdk.operatorframework.io/docs/best-practices/common-recommendation/), Operator SDK Best Practices (https://sdk.operatorframework.io/docs/best-practices/best-practices/)

## What to check

### Idempotency
1. The reconciliation loop MUST be idempotent — calling it multiple times with the same input must produce the same result
2. Use create-or-update (server-side apply or `CreateOrUpdate`) instead of bare Create calls that fail on conflicts
3. Avoid side effects that accumulate on each reconciliation (e.g. creating a new Job every time without checking if one exists)

### Status Management
4. Meaningful status information must be written to CRs at all times — CRs are the primary user interface
5. Use standard Status Conditions following Kubernetes conventions: `type`, `status` (True/False/Unknown), `reason`, `message`, `lastTransitionTime`
6. Status updates should use the status subresource, not the main resource endpoint

### Resource Lifecycle
7. Use finalizers when cleanup logic is needed before resource deletion
8. Implement resource pruning for completed Jobs/Pods — object accumulation consumes etcd storage and slows the API
9. Consider MaxCount or MaxAge strategies for pruning completed resources
10. Use preDelete hooks to extract logs or metrics before pruning

### Versioning and Evolution
11. Use semantic versioning for the operator itself
12. Follow Kubernetes API versioning guidelines for CRDs (v1alpha1 → v1beta1 → v1)
13. Use CRD conversion webhooks to handle older API versions when evolving APIs
14. Operators must support updating operands that were set up by an older version of the operator

## Why it matters

A non-idempotent reconciler contradicts the fundamental controller-runtime design. Resources get stuck in broken states requiring manual intervention. Without proper status, users have no visibility into what the operator is doing. Without pruning, etcd fills up and the entire cluster degrades.

# Review Pattern: Kubernetes Operator Design Principles

**Review-Area**: architecture
**Detection-Hint**: Operator/controller code managing CRDs — look for reconcile loops, multi-CRD controllers, operators deploying other operators, hardcoded namespaces
**Severity**: WARNING
**Category**: technology
**Applies-When**: kubernetes
**Sources**: Operator SDK Best Practices (https://sdk.operatorframework.io/docs/best-practices/best-practices/), Operator SDK Common Recommendations (https://sdk.operatorframework.io/docs/best-practices/common-recommendation/)

## What to check

### Single Responsibility
1. An operator should manage a single type of application — do one thing and do it well
2. Multi-component applications (e.g. Redis + AMQ + MySQL) require separate operators per component
3. A higher-level orchestrating operator may coordinate sub-operators, but should not contain their logic

### CRD Ownership
4. Only one operator should control a CRD on a cluster — multiple operators managing the same CRD is an anti-pattern
5. Each CRD should have its own controller (one reconciliation loop per CRD)
6. A single controller must not reconcile multiple Kinds — this violates encapsulation, SRP, and cohesion

### No Nested Operator Deployment
7. An operator must not deploy or manage other operators — operator lifecycle is the responsibility of a lifecycle manager (e.g. OLM)
8. Operators should not create CRDs in their reconciliation loops — CRDs are global resources requiring careful lifecycle management

### Namespace and Configuration
9. Operators must not hardcode the namespace they are deployed in
10. Operators must not hardcode the namespaces they watch — this should be configurable (empty = watch all)
11. Operators must not hardcode names of resources they expect to already exist
12. No user input should be required to start the operator — it should deploy by deploying its controllers
13. If operator configuration is needed, use a dedicated Configuration CRD (not environment variables or ConfigMaps)

## Why it matters

Operators are long-running controllers that encode operational knowledge. Violating these principles leads to tightly coupled systems where a single operator becomes a monolithic bottleneck, namespace conflicts prevent multi-tenant deployment, and CRD ownership races cause unpredictable behavior.

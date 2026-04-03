# Review Pattern: Kubernetes API Conventions

**Review-Area**: architecture
**Detection-Hint**: Custom Resource Definitions, API types, controller code with non-standard status patterns
**Severity**: WARNING
**Category**: technology
**Applies-When**: kubernetes
**Sources**: Kubernetes API Conventions (https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md), API Changes Guidelines (https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md)

## What to check

1. CRD names follow the convention: `<plural>.<group>` (e.g., `crontabs.stable.example.com`)
2. Status subresource is used for status updates (not the main resource endpoint)
3. Conditions follow standard patterns: `type`, `status` (True/False/Unknown), `reason`, `message`, `lastTransitionTime`
4. Labels follow the recommended format: `app.kubernetes.io/name`, `app.kubernetes.io/version`, etc.
5. Finalizers are used correctly: added before the resource needs protection, removed after cleanup
6. API versions follow stability levels: `v1alpha1` → `v1beta1` → `v1`

## Why it matters

Following API conventions ensures interoperability with the Kubernetes ecosystem (kubectl, dashboards, operators) and reduces surprise for users familiar with native resources.

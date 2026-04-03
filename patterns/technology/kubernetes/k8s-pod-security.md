# Review Pattern: Kubernetes Pod Security

**Review-Area**: security
**Detection-Hint**: Containers running as root, missing securityContext, privileged containers, missing readOnlyRootFilesystem
**Severity**: CRITICAL
**Category**: technology
**Applies-When**: kubernetes, helm
**Sources**: Pod Security Standards (https://kubernetes.io/docs/concepts/security/pod-security-standards/), NSA Kubernetes Hardening Guide (https://media.defense.gov/2022/Aug/29/2003066362/-1/-1/0/CTR_KUBERNETES_HARDENING_GUIDANCE_1.2_20220829.PDF)

## What to check

1. `runAsNonRoot: true` should be set in securityContext
2. `readOnlyRootFilesystem: true` unless the application requires writable filesystem
3. `privileged: false` (or not set) — never run privileged containers
4. `allowPrivilegeEscalation: false` should be explicit
5. Drop all capabilities and add only what is needed: `capabilities: { drop: ["ALL"], add: [...] }`
6. `hostNetwork`, `hostPID`, `hostIPC` should not be true unless absolutely required

## Why it matters

A container running as root with a writable filesystem is one exploit away from node compromise. Defense in depth requires minimizing the attack surface even for trusted workloads.

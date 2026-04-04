# Best Practice Sources

Central catalog of all best-practice guides referenced by review patterns.

## Technology

### Go
| Source | URL | Description |
|--------|-----|-------------|
| Effective Go | https://go.dev/doc/effective_go | Official Go style and idiom guide |
| Go Code Review Comments | https://github.com/golang/go/wiki/CodeReviewComments | Common review points from the Go team |
| Go Proverbs | https://go-proverbs.github.io/ | Rob Pike's design principles for Go |
| Go Blog: Concurrency Patterns | https://go.dev/blog/pipelines | Patterns for goroutines and channels |
| Go Blog: Contexts and structs | https://go.dev/blog/context-and-structs | Proper context.Context usage |

### Kubernetes
| Source | URL | Description |
|--------|-----|-------------|
| Kubernetes API Conventions | https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md | CRD and API design guide |
| Pod Security Standards | https://kubernetes.io/docs/concepts/security/pod-security-standards/ | Pod hardening levels (restricted/baseline/privileged) |
| NSA Kubernetes Hardening Guide | https://media.defense.gov/2022/Aug/29/2003066362/-1/-1/0/CTR_KUBERNETES_HARDENING_GUIDANCE_1.2_20220829.PDF | U.S. government Kubernetes security guide |
| Production Best Practices | https://learnk8s.io/production-best-practices | Comprehensive production readiness checklist |
| CNCF TAG App Delivery: Operator White Paper | https://tag-app-delivery.cncf.io/whitepapers/operator/ | Operator capabilities, maturity model, security, lifecycle, design patterns |
| Operator SDK Best Practices | https://sdk.operatorframework.io/docs/best-practices/best-practices/ | Operator design, CRD ownership, namespace config |
| Operator SDK Common Recommendations | https://sdk.operatorframework.io/docs/best-practices/common-recommendation/ | Idempotent reconciliation, single-Kind controllers |
| Operator SDK Designing Lean Operators | https://sdk.operatorframework.io/docs/best-practices/designing-lean-operators/ | Filtered caches, memory optimization |
| Operator SDK Managing Resources | https://sdk.operatorframework.io/docs/best-practices/managing-resources/ | Resource requests/limits for operators |
| Operator SDK Observability | https://sdk.operatorframework.io/docs/best-practices/observability-best-practices/ | Metrics, alerts, events for operators |
| Operator SDK Multi-Tenancy | https://sdk.operatorframework.io/docs/best-practices/multi-tenancy/ | NetworkPolicy, ingress, namespace isolation |

### Kubernetes Blog
| Source | URL | Description |
|--------|-----|-------------|
| Configuration Good Practices | https://kubernetes.io/blog/2025/11/25/configuration-good-practices/ | YAML conventions, minimal manifests, annotations, grouping |
| 7 Common Kubernetes Pitfalls | https://kubernetes.io/blog/2025/10/20/seven-kubernetes-pitfalls-and-how-to-avoid/ | Resource limits, probes, logging, and common mistakes |
| The Case for Resource Limits | https://kubernetes.io/blog/2023/11/16/the-case-for-kubernetes-resource-limits/ | Predictability vs. efficiency trade-off for limits |
| Start Sidecar First | https://kubernetes.io/blog/2025/06/03/start-sidecar-first/ | Startup probes to sequence sidecar init before main app |
| Multi-Container Pods Overview | https://kubernetes.io/blog/2025/04/22/multi-container-pods-overview/ | Sidecar, ambassador, adapter patterns |
| Protect Pods with PriorityClass | https://kubernetes.io/blog/2023/01/12/protect-mission-critical-pods-priorityclass/ | PriorityClass for critical workload eviction protection |
| 11 Ways Not to Get Hacked | https://kubernetes.io/blog/2018/07/18/11-ways-not-to-get-hacked/ | RBAC, TLS, network policies, non-root, image scanning, YAML analysis |
| Securing Admission Controllers | https://kubernetes.io/blog/2022/01/19/secure-your-admission-controllers-and-webhooks/ | Webhook security hardening |
| NSA/CISA Hardening Guidance Analysis | https://kubernetes.io/blog/2021/10/05/nsa-cisa-kubernetes-hardening-guidance/ | Commentary on NSA hardening guide recommendations |
| Security Best Practices for Deployment | https://kubernetes.io/blog/2016/08/security-best-practices-kubernetes-deployment/ | Image scanning, authorized images, access control, resource quotas |
| Cloud Native Security | https://kubernetes.io/blog/2020/11/18/cloud-native-security-for-your-clusters/ | 4C security model for cloud-native clusters |
| Using Finalizers to Control Deletion | https://kubernetes.io/blog/2021/05/14/using-finalizers-to-control-deletion/ | Finalizer lifecycle, dead finalizer anti-pattern, cleanup |
| Server Side Apply | https://kubernetes.io/blog/2022/10/20/advanced-server-side-apply/ | Field ownership, conflict handling, controller patterns |
| Annotating Services for Humans | https://kubernetes.io/blog/2021/04/20/annotating-k8s-for-humans/ | a8r.io annotation standard, operational metadata |
| Non-root Containers and Devices | https://kubernetes.io/blog/2021/11/09/non-root-containers-and-devices/ | Non-root user patterns with device access |
| Container Design Patterns | https://kubernetes.io/blog/2016/06/container-design-patterns/ | Foundational multi-container design patterns |
| Principles of Container App Design | https://kubernetes.io/blog/2018/03/principles-of-container-app-design/ | Container-native application design principles |
| Enforce CRD Immutability with CEL | https://kubernetes.io/blog/2022/09/29/enforce-immutability-using-cel/ | CEL transition rules for immutable CRD fields |
| Validating Admission Policy (GA) | https://kubernetes.io/blog/2024/04/24/validating-admission-policy-ga/ | Native policy enforcement without webhooks |

### Kubernetes Anti-Patterns & Community
| Source | URL | Description |
|--------|-----|-------------|
| Nutanix: K8s Anti-Patterns for Virtualization Pros | https://www.nutanix.dev/2026/04/01/avoid-these-kubernetes-anti-patterns-a-guide-for-virtualization-professionals/ | Node snapshots, manual SSH, static IPs, cluster-admin everywhere, self-signed certs |
| DevOps.dev: K8s Anti-Patterns Field Guide | https://blog.devops.dev/kubernetes-anti-patterns-a-field-guide-to-cluster-sabotage-a0c8e8969824 | latest tag, root containers, missing limits, secrets in ConfigMaps, single replica, default namespace |
| OneUptime: GitOps Anti-Patterns | https://oneuptime.com/blog/post/2026-02-26-gitops-anti-patterns/view | Secrets in Git, manual kubectl alongside GitOps, monorepo pitfalls, environment promotion |
| Axelspire: Certificate Anti-Patterns in Multi-Cluster K8s | https://axelspire.com/blog/localhost-is-not-an-identity-certificate-anti-patterns-in-multi-cluster-kubernetes/ | Localhost SANs, throwaway self-signed CAs, missing issuance policy, PKI ownership |
| 7 K8s Anti-Patterns That Hurt in Production | https://medium.com/devops-ai-decoded/7-kubernetes-anti-patterns-that-hurt-in-production-91682dbccc5b | Liveness probe misuse with external dependencies, cascading restart failures |
| Kubernetes Operators Deep Dive: Internals | https://dev.to/piyushjajoo/kubernetes-operators-a-deep-dive-into-the-internals-221m | Reconciliation patterns, CRD design, caching, conflict handling, leader election, webhooks |

### Python
| Source | URL | Description |
|--------|-----|-------------|
| PEP 484 - Type Hints | https://peps.python.org/pep-0484/ | Specification for type annotations |
| PEP 3134 - Exception Chaining | https://peps.python.org/pep-3134/ | Exception chain preservation with `from` |
| mypy Documentation | https://mypy.readthedocs.io/ | Static type checker reference |

### Helm
| Source | URL | Description |
|--------|-----|-------------|
| Helm Best Practices | https://helm.sh/docs/chart_best_practices/ | Official chart authoring guide |
| Helm Template Functions | https://helm.sh/docs/chart_template_guide/function_list/ | Template function reference |

## Design Principles

| Source | URL | Description |
|--------|-----|-------------|
| Clean Code | - | Robert C. Martin: SOLID principles, clean architecture |
| Clean Architecture | - | Robert C. Martin: dependency rules, boundaries |
| Extreme Programming Explained | - | Kent Beck: TDD, YAGNI, simple design |
| The Pragmatic Programmer | https://pragprog.com/titles/tpp20/the-pragmatic-programmer-20th-anniversary-edition/ | Hunt/Thomas: DRY principle, pragmatic approaches |
| Test Driven Development: By Example | - | Kent Beck: red-green-refactor cycle |
| Growing Object-Oriented Software, Guided by Tests | - | Freeman/Pryce: outside-in TDD |
| BDD in Action | - | John Ferguson Smart: behavior specifications |
| Design Patterns | - | Gamma et al. (Gang of Four): classic OOP patterns |
| Agile Software Development: Principles, Patterns, and Practices | - | Robert C. Martin: SOLID deep dive |
| A Behavioral Notion of Subtyping | - | Barbara Liskov, Jeannette Wing: original LSP paper |

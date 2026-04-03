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
| Operator SDK Best Practices | https://sdk.operatorframework.io/docs/best-practices/best-practices/ | Operator design, CRD ownership, namespace config |
| Operator SDK Common Recommendations | https://sdk.operatorframework.io/docs/best-practices/common-recommendation/ | Idempotent reconciliation, single-Kind controllers |
| Operator SDK Designing Lean Operators | https://sdk.operatorframework.io/docs/best-practices/designing-lean-operators/ | Filtered caches, memory optimization |
| Operator SDK Managing Resources | https://sdk.operatorframework.io/docs/best-practices/managing-resources/ | Resource requests/limits for operators |
| Operator SDK Observability | https://sdk.operatorframework.io/docs/best-practices/observability-best-practices/ | Metrics, alerts, events for operators |
| Operator SDK Multi-Tenancy | https://sdk.operatorframework.io/docs/best-practices/multi-tenancy/ | NetworkPolicy, ingress, namespace isolation |

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

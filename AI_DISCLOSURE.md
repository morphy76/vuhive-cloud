# Spec-Driven Development (SDD) & AI Agent Adoption Disclosure

Welcome to **`vuhive-cloud`**. This document provides a transparent disclosure of our software engineering philosophy: `vuhive-cloud` is built using **Spec-Driven Development (SDD)** and actively embraces an **agent-native engineering paradigm**.

We believe in radical transparency regarding the role of Artificial Intelligence in our codebase. Rather than using AI as an opaque autocomplete assistant or generating unvetted code, this project operates on a rigorous partnership between human architects and autonomous coding agents governed by deterministic specifications and mechanical verification gates.

---

## 1. Core Tenets & Philosophy of Spec-Driven Development (SDD)

Spec-Driven Development (SDD) is the foundational engineering discipline of `vuhive-cloud`. In traditional software development, design decisions are often dispersed across informal chat threads, stale tickets, or implicit developer memory, leaving AI tools to guess architectural intent. SDD shifts the center of gravity to **formal, executable, and living specifications**.

### The Human "What" vs. Agent "How" Division

We strictly separate engineering responsibilities between humans and AI agents:

```text
┌───────────────────────────────────────────────────────────────────────────────────┐
│ HUMAN ARCHITECT (The "What")                                                      │
│                                                                                   │
│  • Problem Definition & Business Invariants                                       │
│  • Domain Boundaries & Hexagonal Port Contracts                                   │
│  • Architectural Specifications (ARCHITECTURE_SPEC.md)                            │
│  • System Rules & Constraints (.agents/rules/)                                    │
│  • Acceptance Criteria & PR Review Approval                                       │
└────────────────────────────────────────┬──────────────────────────────────────────┘
                                         │ Governs & Steers
                                         ▼
┌───────────────────────────────────────────────────────────────────────────────────┐
│ AUTONOMOUS CODING AGENTS (The "How")                                              │
│                                                                                   │
│  • Translates Specifications into Idiomatic Go Code                               │
│  • Executes Strict TDD Cycles (Red -> Green -> Refactor)                          │
│  • Generates Edge-Case Test Harnesses & Mocks                                     │
│  • Handles Boilerplate, DTO Mappings & Adapters                                   │
│  • Implements Mechanical Refactoring & Code Quality Fixes                         │
└────────────────────────────────────────┬──────────────────────────────────────────┘
                                         │ Subject To
                                         ▼
┌───────────────────────────────────────────────────────────────────────────────────┐
│ DETERMINISTIC QUALITY GATES (Mechanical Verification)                             │
│                                                                                   │
│  • Compile-Time Interface Assertions (var _ Port = (*Adapter)(nil))               │
│  • Go 1.26 Strict Compiler & golangci-lint                                        │
│  • Race Detector (-race) & Test Coverage                                          │
│  • Kubernetes Pod Security Standard Verification (Restricted Profile)             │
│  • Mandatory Human Pull Request Review & Merge Approval                           │
└───────────────────────────────────────────────────────────────────────────────────┘
```

1. **Humans Envision and Specify the "What"**:
   - The human acts as the system architect, product owner, and quality guardian.
   - Humans formulate domain models, state machines, concurrency expectations, security invariants (e.g., Kubernetes restricted security profiles), and acceptance criteria.
   - Humans author and maintain the authoritative architectural specification ([`ARCHITECTURE_SPEC.md`](./ARCHITECTURE_SPEC.md)) and repository governance rules ([`.agents/rules/`](./.agents/rules/)).

2. **Agents Implement and Execute the "How"**:
   - Autonomous coding agents (such as Google DeepMind Antigravity, Claude, Cursor, or OpenAI Codex) serve as tireless pair programmers and execution engines.
   - Agents ingest the specifications, explore the codebase context, write comprehensive failing test cases first (TDD), implement the business and infrastructure logic, and perform mechanical refactoring.
   - Agents operate under bounded workspace environments with deterministic tools (compilers, linters, test runners).

---

## 2. Architecture & Specification Alignment

Autonomous coding agents are only as reliable as the constraints that steer them. In `vuhive-cloud`, agents do not operate unconstrained; they are strictly steered by living specifications and declarative rule sets.

### The Rulebook (`.agents/rules/`)

The repository maintains an executable `.agents/` rulebook that every coding agent must ingest and enforce without exception:

| Rule File | Governed Principles & Constraints |
|---|---|
| [`.agents/rules/code-architecture.md`](./.agents/rules/code-architecture.md) | **Hexagonal Architecture (Ports & Adapters) & Domain-Driven Design (DDD)**:<br>• Pure domain layer (`domain/`) has zero dependencies on application or adapters.<br>• Application layer (`application/`) depends only on domain and port interfaces (`ports/inbound`, `ports/outbound`).<br>• Infrastructure layer (`adapters/`) implements outbound ports and drives inbound ports.<br>• Reactive Go concurrency (goroutines, worker pools, channels, context cancellation, no unbounded concurrency). |
| [`.agents/rules/golang.md`](./.agents/rules/golang.md) | **Static Compile-Time Interface Verification**:<br>• Mandatory static type assertions (`var _ Port = (*Adapter)(nil)`) on every adapter struct.<br>• Minimal, focused interfaces ("accept interfaces, return structs"). |
| [`.agents/rules/tdd.md`](./.agents/rules/tdd.md) | **Strict Test-Driven Development (TDD)**:<br>• Mandatory Red-Green-Refactor cycle.<br>• Zero production code introduced without a prior failing test.<br>• Continuous regression verification. |
| [`.agents/rules/nonfunctional.md`](./.agents/rules/nonfunctional.md) | **Non-Functional Requirements & Stack Lock-in**:<br>• Go 1.26, Gin, `zerolog` structured logging (no unstructured format strings, mandatory enter/exit log patterns with duration ms), Viper, `testify`.<br>• Monorepo root `Makefile` with atomic compile-time version injection via `ldflags`.<br>• Graceful teardown listening for OS termination signals. |
| [`.agents/rules/k8s-local-validation.md`](./.agents/rules/k8s-local-validation.md) | **Local Kubernetes Validation Safety (Rancher Desktop)**:<br>• Context verification locked to `rancher-desktop`.<br>• Isolated ephemeral namespaces (`vuhive-smoke-*`) with guaranteed post-validation teardown.<br>• BuildKit `--load` image containment and in-cluster probing.<br>• Robust probe file-staging patterns (Base64 stdin pipeline avoiding container `tar` dependencies). |

### Architectural Spec (`ARCHITECTURE_SPEC.md`)

[`ARCHITECTURE_SPEC.md`](./ARCHITECTURE_SPEC.md) acts as the single source of truth for:
- Bounded contexts, aggregates, and domain state machines (e.g. `TestSuite`, `Artifact`, `RunnerProfile`, `Schedule`, `TestRun`, `BarrierSession`).
- Database schema (DDL for PostgreSQL via Goose migrations).
- Pod Security Standards (`Restricted` profile with non-root UID `10001`, read-only root filesystems, dropped capabilities).
- Distributed coordination protocols (rendezvous barrier and workload partitioning).

---

## 3. Human-in-the-Loop Governance & Quality Gates

AI-generated code is **never deployed unchecked**. Every line of code written by an agent must clear multi-layered automated and human verification gates before landing on `main`:

```text
  Agent Output
       │
       ▼
┌─────────────────────────────────────────────────────────┐
│ 1. Compile-Time Type Assertion Gate                     │  -> Static verification of interface satisfaction
│    (var _ Interface = (*Concrete)(nil))                 │
└──────────────────────────────┬──────────────────────────┘
                               │ PASS
                               ▼
┌─────────────────────────────────────────────────────────┐
│ 2. Automated Test & Coverage Gate (TDD)                 │  -> go test -v -race ./...
│    (Red -> Green -> Refactor verification)              │
└──────────────────────────────┬──────────────────────────┘
                               │ PASS
                               ▼
┌─────────────────────────────────────────────────────────┐
│ 3. Static Analysis & Linting Gate                       │  -> golangci-lint, go vet, deadcode detection
│    (Clean code, SOLID, zero unhandled errors)           │
└──────────────────────────────┬──────────────────────────┘
                               │ PASS
                               ▼
┌─────────────────────────────────────────────────────────┐
│ 4. Local Cluster Smoke / Sandbox Gate                   │  -> Ephemeral K8s namespace validation
│    (When infrastructure changes occur)                  │
└──────────────────────────────┬──────────────────────────┘
                               │ PASS
                               ▼
┌─────────────────────────────────────────────────────────┐
│ 5. CI / GitHub Actions Validation Pipeline              │  -> Multi-platform build, unit & integration tests
└──────────────────────────────┬──────────────────────────┘
                               │ PASS
                               ▼
┌─────────────────────────────────────────────────────────┐
│ 6. HUMAN ARCHITECT CODE REVIEW & SIGN-OFF               │  -> Final PR inspection, design approval, merge
└─────────────────────────────────────────────────────────┘
```

1. **Static Type Checking**: The Go compiler verifies static interface satisfaction, memory safety, and type boundaries. If an agent fails to implement an interface method or introduces an invalid type, compilation fails immediately.
2. **Deterministic Test Suites**: Tests are not an afterthought. Unit and integration tests (using `testify` and `testcontainers-go`) verify behavior across boundary conditions, error flows, and concurrency limits. Tests must pass with the Go race detector enabled (`go test -race`).
3. **Structured Logging & Observability**: Every component adheres to structured logging with `zerolog`. Log entries have typed fields (no `fmt.Sprintf` interpolation) and follow enter/exit tracking with execution durations.
4. **Human Review & Final Authority**: Pull requests are reviewed by human maintainers. Humans evaluate architectural elegance, long-term maintainability, security postures, and adherence to system goals. Agents do not merge their own pull requests without human oversight.

---

## 4. Contributor Guidelines with AI Tools

We encourage contributors to leverage modern coding agents (e.g., Antigravity, Claude, Cursor, GitHub Copilot, Gemini) when contributing to `vuhive-cloud`. To ensure seamless collaboration and pristine code quality, follow these guidelines:

### A. Ground Agents in the Specification First
- Always direct your agent to read [`ARCHITECTURE_SPEC.md`](./ARCHITECTURE_SPEC.md) and the relevant rules in [`.agents/rules/`](./.agents/rules/) before prompting it to generate code.
- Provide the exact GitHub Issue description and acceptance criteria to the agent.
- Keep agent focus scoped: tackle one issue or component at a time rather than broad, unfocused refactors.

### B. Enforce Test-Driven Development (TDD)
- Instruct your agent to write a failing test first ([`.agents/rules/tdd.md`](./.agents/rules/tdd.md)).
- Run the test to observe failure (`go test -v ./internal/...`).
- Have the agent implement the minimal code required to pass, then refactor for readability and performance.

### C. Maintain Layering & Static Interface Checks
- When implementing a new repository or external client, declare the interface in `internal/<component>/application/ports/outbound/`.
- Implement the concrete adapter in `internal/<component>/adapters/outbound/<infra>/`.
- Always add the static interface assertion at declaration time:
  ```go
  var _ outbound.MyPort = (*MyAdapter)(nil)
  ```
- Ensure domain entities never import adapters or framework tags (`json:`, `db:`).

### D. Run Automated Verification Locally
Before submitting a pull request, run the verification suite:
```bash
# 1. Run unit tests with race detection
make test-race

# 2. Run linter
make lint

# 3. Build all binaries
make build
```

### E. Author Clear, Auditable Commits
- Follow Conventional Commits format (e.g. `feat(orchestration): ... (closes #X)`, `docs(project): ...`).
- In pull request descriptions, disclose which agentic tools were utilized and link the corresponding issue and milestone.

---

## Summary

`vuhive-cloud` demonstrates that AI agents and rigorous software engineering do not compete—they amplify each other. By anchoring autonomous agents with deep specifications, strict architectural rules, and uncompromising automated quality gates, we build higher-quality, better-tested, and more resilient software at high velocity.

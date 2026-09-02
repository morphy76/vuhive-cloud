# vuhive-cloud Project Documentation & Setup Package

This directory contains the architecture specification, system design, and milestone/issue breakdown for the future **`vuhive-cloud`** project (`github.com/morphy76/vuhive-cloud`).

## Documents in this Package

1. **[ARCHITECTURE_SPEC.md](./ARCHITECTURE_SPEC.md)**
   - **Executive Summary & System Vision**
   - **System Architecture & Topology Diagram** (Control Plane, PostgreSQL, S3/MinIO, K8s Ephemeral Build Jobs, Runner Pods, Node Affinities/Tolerations)
   - **Hexagonal Architecture & Package Layout** (DDD Boundaries, Domain Aggregates, Inbound/Outbound Ports)
   - **Detailed Execution Workflows** (Source Upload -> Ephemeral Compilation -> S3 Storage -> K8s Runner Job / Native CronJob -> Ingestion)
   - **PostgreSQL Database Schema (DDL)** (`test_suites`, `artifacts`, `configurations`, `runner_profiles`, `schedules`, `test_runs`)
   - **REST API Contract & Endpoints**
   - **Kubernetes Runner Pod Specification** (Init-container artifact fetch, emptyDir mount, execution wrapper)

2. **[ROADMAP_AND_GITHUB_ISSUES.md](./ROADMAP_AND_GITHUB_ISSUES.md)**
   - **Milestone 1: Core Foundation & Single-Runner Cloud Engine**
     - Epic 1.1: Hexagonal Architecture, Domain Models & Data Layer (Issues 1.1.1 – 1.1.3)
     - Epic 1.2: Source-to-Binary Compilation Subsystem (Issues 1.2.1 – 1.2.2)
     - Epic 1.3: Runner Pod Orchestration & Profiles (Issues 1.3.1 – 1.3.3)
     - Epic 1.4: Scheduling, Reporting & CLI (Issues 1.4.1 – 1.4.3)
     - Epic 1.5: Helm Deployment & Infrastructure Packaging (Issue 1.5.1)
   - **Milestone 2: Distributed Multi-Pod Coordination & Live Streaming**
     - Epic 2.1: Distributed Multi-Pod Coordination (Issues 2.1.1 – 2.1.3)
     - Epic 2.2: Live Telemetry Streaming (Issues 2.2.1 – 2.2.2)
   - **Milestone 3: Multi-Namespace, Multi-Cluster & Enterprise SSO**
     - Epic 3.1: Multi-Namespace & Multi-Cluster Dispatcher (Issues 3.1.1 – 3.1.2)
     - Epic 3.2: Enterprise Authentication & RBAC (Issues 3.2.1 – 3.2.2)

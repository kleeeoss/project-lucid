# Lucid-CI

AI-Native DevSecOps Platform for real-time Pull Request vulnerability analysis and SLM-driven remediation.

## Monorepo Layout
- `/platform` — Ingestion Gateway, SQS Worker Pool, PostgreSQL State, GitHub App, Next.js 14 Dashboard (Owner: Krish)
- `/engine` — Tree-sitter AST Parser, Intraprocedural Taint Graph, Static Rule Catalog (Owner: Apurv)
- `/infra-ai` — Python FastAPI Remediation Microservice, Ephemeral gVisor Detonation Sandbox, Terraform IaC (Owner: Garv)
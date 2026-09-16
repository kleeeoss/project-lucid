# Lucid-CI — Infra & AI Microservice (`/infra-ai`)

**Domain Owner:** Garv (AI & Cloud Architect)
**Service Stack:** Python 3.12+, FastAPI, Pydantic V2, Groq / Gemini SLMs, Docker (gVisor), Terraform

## Subsystem Overview
This service provides two core DevSecOps capabilities:
1. **AI-Assisted Remediation (`service/`):** Generates structured JSON security fixes using nonce-delimited prompts and hosted SLMs.
2. **Ephemeral Dynamic Sandboxing (`sandbox/`):** Detonates untrusted PR code in gVisor containers with absolute network isolation (`--network none`).

## Local Setup (WSL2 / Linux)

```bash
# From the repo root
cd infra-ai

# Create and activate virtual environment
python3 -m venv .venv
source .venv/bin/activate

# Upgrade pip and install dependencies
pip install --upgrade pip
pip install -r requirements.txt

# Run unit test suite
pytest -v tests/
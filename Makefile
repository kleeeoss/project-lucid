.PHONY: test test-all test-engine test-platform test-schemas test-pipeline test-local demo-up demo-down demo-status build-docker

# Master Test Target
test-all: test-engine test-platform test

# Go Engine & Platform Unit Testing
test-engine:
	@echo "Running Engine test suite..."
	@cd engine && go test -v -race ./...

test-platform:
	@echo "Running Platform test suite..."
	@cd platform && go test -v -race ./...

# Python AI Service Unit Testing
test:
	@cd infra-ai && if [ -f .venv/bin/pytest ]; then .venv/bin/pytest -v; else pytest -v; fi

test-schemas:
	@cd infra-ai && if [ -f .venv/bin/pytest ]; then .venv/bin/pytest -v tests/test_schemas.py; else pytest -v tests/test_schemas.py; fi

test-pipeline:
	@cd infra-ai && if [ -f .venv/bin/pytest ]; then .venv/bin/pytest -v tests/test_remediation_pipeline.py; else pytest -v tests/test_remediation_pipeline.py; fi


test-local:
	@bash scripts/test_e2e_local.sh

# Container Builds
build-docker:
	@docker compose -f docker-compose.prod.yml build

# On-Demand AWS Showcase Lifecycle Targets
demo-up:
	@bash scripts/demo.sh up $(RUNTIME)

demo-down:
	@bash scripts/demo.sh down

demo-status:
	@bash scripts/demo.sh status

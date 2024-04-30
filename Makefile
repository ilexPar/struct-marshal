.ONESHELL:

SHELL := /bin/bash
STATIC_ANALYSIS_FILE := analysis-report.html

###### Development ######

test::
	@gotestsum -f dots-v2 -- -coverprofile cover.out -coverpkg=./pkg/... ./tests/... 

test-html:: test
	@go tool cover -html=cover.out

analyse::
	@echo "Performing Static Analysis with golangci-lint"
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:v1.57 golangci-lint run --out-format html --tests=false --timeout 5m0s > ${STATIC_ANALYSIS_FILE}
	firefox ${STATIC_ANALYSIS_FILE}
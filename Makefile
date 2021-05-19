GIT_COMMIT ?= $(if $(shell git rev-parse --short HEAD),$(shell git rev-parse --short HEAD),$(shell cat ./git-commit | head -c 7))
SOURCE_DATE_EPOCH ?= $(if $(shell date +%s),$(shell date +%s),$(error "date failed"))
VERSION := $(shell cat ./VERSION)
ARCH := $(shell arch)

test: test-unit

.PHONY: test-unit
test-unit:
	@echo "Unit tests starting..."
	@go test -race -cover -count=1 -timeout=300s  ./...
	@echo "Units test done!"

.PHONY: clean
clean:
	@echo "clean...."
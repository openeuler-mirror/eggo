GIT_COMMIT ?= $(if $(shell git rev-parse --short HEAD),$(shell git rev-parse --short HEAD),$(shell cat ./git-commit | head -c 7))
SOURCE_DATE_EPOCH ?= $(if $(shell date +%s),$(shell date +%s),$(error "date failed"))
VERSION := $(shell cat ./VERSION)
ARCH := $(shell arch)

.PHONY: eggo
eggo:
	@echo "build eggo starting..."
	@go build -buildmode=pie -ldflags '-extldflags=-static' -ldflags '-linkmode=external -extldflags=-Wl,-z,relro,-z,now' -o eggo ./cmd/
	@echo "build eggo done!"
test:
	@echo "Unit tests starting..."
	@go test -race -cover -count=1 -timeout=300s  ./...
	@echo "Units test done!"

.PHONY: clean
clean:
	@echo "clean...."
	@rm -f eggo
	@echo "clean done!"

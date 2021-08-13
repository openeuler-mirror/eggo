GIT_COMMIT ?= $(if $(shell git rev-parse --short HEAD),$(shell git rev-parse --short HEAD),$(shell cat ./git-commit | head -c 7))
SOURCE_DATE_EPOCH ?= $(if $(shell date +%s),$(shell date +%s),$(error "date failed"))
VERSION := $(shell cat ./VERSION)
ARCH := $(shell arch)

.PHONY: eggo
eggo:
	@echo "build eggo starting..."
	@go build -buildmode=pie -ldflags '-extldflags=-static' -ldflags '-linkmode=external -extldflags=-Wl,-z,relro,-z,now' -o bin/eggo .
	@echo "build eggo done!"
local:
	@echo "build eggo use vendor starting..."
	@go build -buildmode=pie -ldflags '-extldflags=-static' -mod vendor -ldflags '-linkmode=external -extldflags=-Wl,-z,relro,-z,now' -o bin/eggo .
	@echo "build eggo use vendor done!"
test:
	@echo "Unit tests starting..."
	@go test -race -cover -count=1 -timeout=300s  ./...
	@echo "Units test done!"

images: image-eggo

image-eggo: eggo
	cp bin/eggo images/eggo/ && \
	docker build -t eggo:$(VERSION) images/eggo && \
	rm images/eggo/eggo

.PHONY: install
install:
	@echo "install eggo..."
	@install -d /usr/local/bin
	@install -m 0750 bin/eggo /usr/local/bin
	@echo "install eggo done"

.PHONY: clean
clean:
	@echo "clean...."
	@rm -rf ./bin
	@echo "clean done!"

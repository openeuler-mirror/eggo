GIT_COMMIT ?= $(if $(shell git rev-parse --short HEAD),$(shell git rev-parse --short HEAD),$(error "commit id failed"))
SOURCE_DATE_EPOCH ?= $(if $(shell date +%s),$(shell date +%s),$(error "date failed"))
VERSION := $(shell cat ./VERSION)
ARCH := $(shell arch)

EXTRALDFLAGS :=
LDFLAGS := -X isula.org/eggo/cmd.Version=$(VERSION) \
		   -X isula.org/eggo/cmd.Commit=$(GIT_COMMIT) \
		   -X isula.org/eggo/cmd.BuildTime=$(SOURCE_DATE_EPOCH) \
		   -X isula.org/eggo/cmd.Arch=$(ARCH) \
		   $(EXTRALDFLAGS)
SAFEBUILDFLAGS := -buildmode=pie -extldflags=-ftrapv -extldflags=-zrelro -extldflags=-znow -tmpdir=/tmp/xxeggo $(LDFLAGS)

.PHONY: eggo
eggo:
	@echo "build eggo starting..."
	@go build -ldflags '$(LDFLAGS)' -o bin/eggo .
	@echo "build eggo done!"
local:
	@echo "build eggo use vendor starting..."
	@go build -ldflags '$(LDFLAGS)' -mod vendor -ldflags -o bin/eggo .
	@echo "build eggo use vendor done!"
test:
	@echo "Unit tests starting..."
	@go test -race -cover -count=1 -timeout=300s  ./...
	@echo "Units test done!"

.PHONY: safe
safe:
	@echo "build safe eggo starting..."
	go build -ldflags '$(SAFEBUILDFLAGS)' -o bin/eggo .
	@echo "build safe eggo done!"

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

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
STATIC_LDFLAGS := -extldflags=-static -linkmode=external
SAFEBUILDFLAGS := -buildmode=pie -extldflags=-ftrapv -extldflags=-zrelro -extldflags=-znow -tmpdir=/tmp/xxeggo $(LDFLAGS)

GO := go
GO_BUILD := CGO_ENABLED=0 $(GO)

.PHONY: eggo
eggo:
	@echo "build eggo starting..."
	@$(GO_BUILD) build -ldflags '$(LDFLAGS) $(STATIC_LDFLAGS)' -o bin/eggo .
	@echo "build eggo done!"
local:
	@echo "build eggo use vendor starting..."
	@$(GO_BUILD) build -ldflags '$(LDFLAGS) $(STATIC_LDFLAGS)' -mod vendor -o bin/eggo .
	@echo "build eggo use vendor done!"
test:
	@echo "Unit tests starting..."
	@$(GO) test $(shell go list ./... | grep -v /eggops) -race -cover -count=1 -timeout=300s
	@echo "Units test done!"

.PHONY: safe
safe:
	@echo "build safe eggo starting..."
	$(GO_BUILD) build -ldflags '$(SAFEBUILDFLAGS) $(STATIC_LDFLAGS)' -o bin/eggo .
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

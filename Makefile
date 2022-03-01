GIT_COMMIT ?= $(if $(shell git rev-parse --short HEAD),$(shell git rev-parse --short HEAD),$(error "commit id failed"))
SOURCE_DATE_EPOCH ?= $(if $(shell date +%s),$(shell date +%s),$(error "date failed"))
VERSION := $(shell cat ./VERSION)
# eggo arch amd64/arm64
ifndef ARCH
ARCH = amd64
ifeq ($(shell uname -p), aarch64)
ARCH = arm64
endif
endif

EXTRALDFLAGS :=
LDFLAGS := -X isula.org/eggo/cmd.Version=$(VERSION) \
		   -X isula.org/eggo/cmd.Commit=$(GIT_COMMIT) \
		   -X isula.org/eggo/cmd.BuildTime=$(SOURCE_DATE_EPOCH) \
		   -X isula.org/eggo/cmd.Arch=$(ARCH) \
		   $(EXTRALDFLAGS)
STATIC_LDFLAGS := -extldflags=-static -linkmode=external
SAFEBUILDFLAGS := -buildmode=pie -extldflags=-ftrapv -extldflags=-zrelro -extldflags=-znow -tmpdir=/tmp/xxeggo $(LDFLAGS)

GO := go
GO_BUILD := CGO_ENABLED=0 GOARCH=$(ARCH) $(GO)

.PHONY: eggo
eggo:
	@echo "build eggo of $(ARCH) starting..."
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

check:
	@which ${GOPATH}/bin/golangci-lint > /dev/null || (echo "Installing golangci-lint" && go get -d github.com/golangci/golangci-lint/cmd/golangci-lint)
	@echo "Code check starting..."
	@${GOPATH}/bin/golangci-lint run --timeout 5m --config=./.golangci.yaml
	@echo "Code check done!"

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

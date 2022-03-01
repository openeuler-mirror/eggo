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

TMP_PATH := /tmp/xxeggo

EXTRALDFLAGS :=
LDFLAGS := -X isula.org/eggo/cmd.Version=$(VERSION) \
		   -X isula.org/eggo/cmd.Commit=$(GIT_COMMIT) \
		   -X isula.org/eggo/cmd.BuildTime=$(SOURCE_DATE_EPOCH) \
		   -X isula.org/eggo/cmd.Arch=$(ARCH) \
		   $(EXTRALDFLAGS)
STATIC_LDFLAGS := -extldflags=-static -linkmode=external
SAFEBUILDFLAGS := -buildmode=pie \
				  -extldflags=-ftrapv -extldflags=-zrelro -extldflags=-znow \
				  -linkmode=external \
				  -extldflags "-static-pie -Wl,-z,now" \
				  -tmpdir=$(TMP_PATH) \
				  $(LDFLAGS)

GO := go
GO_BUILD := CGO_ENABLED=1 GOARCH=$(ARCH) $(GO)
GO_SAFE_BUILD:= CGO_ENABLE=1 \
				CGO_CFLAGS="-fstack-protector-strong -fPIE" \
				CGO_CPPFLAGS="-fstack-protector-strong -fPIE" \
				CGO_LDFLAGS_ALLOW="-Wl,-z,relro,-z,now" \
				CGO_LDFLAGS="-Wl,-z,relro,-z,now -Wl,-z,noexecstack" \
				GOARCH=$(ARCH) \
				$(GO)

.PHONY: eggo
eggo:
	@echo "build eggo of $(ARCH) starting..."
	@$(GO_BUILD) build -ldflags '$(LDFLAGS) $(STATIC_LDFLAGS)' -o bin/eggo . 2>/dev/null
	@echo "build eggo done!"
local:
	@echo "build eggo use vendor starting..."
	@$(GO_BUILD) build -ldflags '$(LDFLAGS) $(STATIC_LDFLAGS)' -mod vendor -o bin/eggo . 2>/dev/null
	@echo "build eggo use vendor done!"
test:
	@echo "Unit tests starting..."
	@$(GO) test -race -cover -count=1 -timeout=300s  ./...
	@echo "Units test done!"

check:
	@which ${GOPATH}/bin/golangci-lint > /dev/null || (echo "Installing golangci-lint" && go get -d github.com/golangci/golangci-lint/cmd/golangci-lint)
	@echo "Code check starting..."
	@${GOPATH}/bin/golangci-lint run --timeout 5m --config=./.golangci.yaml
	@echo "Code check done!"

.PHONY: safe
safe:
	@echo "build safe eggo starting..."
	mkdir -p $(TMP_PATH)
	$(GO_SAFE_BUILD) build -ldflags '$(SAFEBUILDFLAGS)' -o bin/eggo .
	rm -rf $(TMP_PATH)
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

# Image URL to use all building/pushing image targets
IMG ?= kubeshop/kube-webhook-certgen:latest
# Cockroach build rules.
GO ?= go
# Allow setting of go build flags from the command line.
GOFLAGS :=
# Set to 1 to use static linking for all builds (including tests).
STATIC :=

ifeq ($(STATIC),1)
LDFLAGS += -s -w -extldflags "-static"
endif

.PHONY: lint
lint: # lint code using golangci-lint
	golangci-lint run --max-issues-per-linter=0 --sort-results ./...

.PHONY: test
test: # run tests using gotestsum
	gotestsum ./...

.PHONY: build
build:
	$(GO) build -ldflags '$(LDFLAGS)' -v -o kube-webhook-certgen

.PHONY: clean
clean:
	rm kube-webhook-certgen

.PHONY: build-macos-apple
build-macos-apple: clean # build executable
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags '$(LDFLAGS)' -v -o kube-webhook-certgen

.PHONY: build-macos-intel
build-macos-intel: clean # build executable
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -v -o kube-webhook-certgen

.PHONY: build-linux
build-linux: clean # build executable for amd64
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -v -o kube-webhook-certgen

.PHONY: docker-build
docker-build: test build ## Run tests and build docker image.
	docker build -t ${IMG} .

.PHONY: docker-build-quick
docker-build-quick: build ## Build docker image.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

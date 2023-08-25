LOCALBIN ?= $(shell pwd)/bin

CONTROLLER_TOOLS_VERSION ?= v0.12.0
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

SRCS := $(shell find . -type f -name '*.go' -not -path "./vendor/*")

$(LOCALBIN)/airgapify: $(LOCALBIN) $(SRCS)
	@mkdir -p bin
	go build -o $@ cmd/airgapify/main.go

generate: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	$(CONTROLLER_GEN) crd output:crd:artifacts:config=deploy paths="./..."

tidy:
	go mod tidy
	go fmt ./...

lint:
	golangci-lint run ./...

test:
	go test -coverprofile=coverage.out -v ./...

clean:
	-rm -rf bin
	go clean -testcache

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: generate tidy lint test clean
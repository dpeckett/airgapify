VERSION 0.7
FROM golang:1.21-bookworm
WORKDIR /app

all:
  COPY (+build/airgapify --GOARCH=amd64) ./dist/airgapify-linux-amd64
  COPY (+build/airgapify --GOARCH=arm64) ./dist/airgapify-linux-arm64
  COPY (+build/airgapify --GOOS=darwin --GOARCH=amd64) ./dist/airgapify-darwin-amd64
  COPY (+build/airgapify --GOOS=darwin --GOARCH=arm64) ./dist/airgapify-darwin-arm64
  RUN cd dist && find . -type f -exec sha256sum {} \; >> ../checksums.txt
  SAVE ARTIFACT ./dist/airgapify-linux-amd64 AS LOCAL dist/airgapify-linux-amd64
  SAVE ARTIFACT ./dist/airgapify-linux-arm64 AS LOCAL dist/airgapify-linux-arm64
  SAVE ARTIFACT ./dist/airgapify-darwin-amd64 AS LOCAL dist/airgapify-darwin-amd64
  SAVE ARTIFACT ./dist/airgapify-darwin-arm64 AS LOCAL dist/airgapify-darwin-arm64
  SAVE ARTIFACT ./checksums.txt AS LOCAL dist/checksums.txt

build:
  ARG GOOS=linux
  ARG GOARCH=amd64
  COPY go.mod go.sum ./
  RUN go mod download
  COPY . .
  RUN CGO_ENABLED=0 go build --ldflags '-s' -o airgapify cmd/main.go
  SAVE ARTIFACT ./airgapify AS LOCAL dist/airgapify-${GOOS}-${GOARCH}

generate:
  ARG CONTROLLER_TOOLS_VERSION=v0.12.0
  RUN go install sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_TOOLS_VERSION}
  COPY . ./
  RUN controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./api/..."
	RUN controller-gen crd output:crd:artifacts:config=dist paths="./api/..."
  SAVE ARTIFACT ./api/v1alpha1/zz_generated.deepcopy.go AS LOCAL api/v1alpha1/zz_generated.deepcopy.go
  SAVE ARTIFACT ./dist/airgapify.pecke.tt_configs.yaml AS LOCAL dist/airgapify.pecke.tt_configs.yaml

tidy:
  LOCALLY
  RUN go mod tidy
  RUN go fmt ./...

lint:
  FROM golangci/golangci-lint:v1.55.2
  WORKDIR /app
  COPY . ./
  RUN golangci-lint run --timeout 5m ./...

test:
  COPY . ./
  RUN go test -coverprofile=coverage.out -v ./...
  SAVE ARTIFACT ./coverage.out AS LOCAL coverage.out
VERSION 0.8
FROM golang:1.22-bookworm
ENV DO_NOT_TRACK=1
WORKDIR /workspace

all:
  ARG VERSION=dev
  BUILD +generate
  COPY (+build/airgapify --GOARCH=amd64) ./dist/airgapify-linux-amd64
  COPY (+build/airgapify --GOARCH=arm64) ./dist/airgapify-linux-arm64
  COPY (+build/airgapify --GOARCH=riscv64) ./dist/airgapify-linux-riscv64
  COPY (+build/airgapify --GOOS=darwin --GOARCH=amd64) ./dist/airgapify-darwin-amd64
  COPY (+build/airgapify --GOOS=darwin --GOARCH=arm64) ./dist/airgapify-darwin-arm64
  COPY (+build/airgapify --GOOS=windows --GOARCH=amd64) ./dist/airgapify-windows-amd64.exe
  COPY (+package/*.deb --GOARCH=amd64) ./dist/
  COPY (+package/*.deb --GOARCH=arm64) ./dist/
  COPY (+package/*.deb --GOARCH=riscv64) ./dist/
  RUN cd dist && find . -type f | sort | xargs sha256sum >> ../sha256sums.txt
  SAVE ARTIFACT ./dist/* AS LOCAL dist/
  SAVE ARTIFACT ./sha256sums.txt AS LOCAL dist/sha256sums.txt

build:
  ARG GOOS=linux
  ARG GOARCH=amd64
  COPY go.mod go.sum ./
  RUN go mod download
  COPY . .
  ARG VERSION=dev
  RUN CGO_ENABLED=0 go build --ldflags "-s -X 'github.com/dpeckett/airgapify/internal/constants.Version=${VERSION}'" -o airgapify main.go
  SAVE ARTIFACT ./airgapify AS LOCAL dist/airgapify-${GOOS}-${GOARCH}

generate:
  ARG CONTROLLER_TOOLS_VERSION=v0.16.2
  RUN go install sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_TOOLS_VERSION}
  COPY . ./
  RUN controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./api/..."
	RUN controller-gen crd output:crd:artifacts:config=dist paths="./api/..."
  SAVE ARTIFACT ./api/v1alpha1/zz_generated.deepcopy.go AS LOCAL api/v1alpha1/zz_generated.deepcopy.go
  SAVE ARTIFACT ./dist/airgapify.pecke.tt_configs.yaml AS LOCAL dist/airgapify.pecke.tt_configs.yaml

tidy:
  LOCALLY
  ENV GOTOOLCHAIN=go1.22.1
  RUN go mod tidy
  RUN go fmt ./...

lint:
  FROM golangci/golangci-lint:v1.59.1
  WORKDIR /workspace
  COPY . ./
  RUN golangci-lint run --timeout 5m ./...

test:
  COPY go.mod go.sum ./
  RUN go mod download
  COPY . .
  RUN go test -coverprofile=coverage.out -v ./...
  SAVE ARTIFACT ./coverage.out AS LOCAL coverage.out

integration-test:
  RUN apt update
  RUN apt install -y jq
  COPY +build/airgapify ./airgapify
  COPY testdata ./testdata
  RUN ./airgapify -f testdata/prometheus.yaml
  RUN [ "$(tar -xf images.tar -O index.json | jq '.manifests | length')" -eq 6 ] || exit 1

package:
  FROM debian:bookworm
  # Use bookworm-backports for newer golang versions
  RUN echo "deb http://deb.debian.org/debian bookworm-backports main" > /etc/apt/sources.list.d/backports.list
  RUN apt update
  # Tooling
  RUN apt install -y git curl devscripts dpkg-dev debhelper-compat git-buildpackage libfaketime dh-sequence-golang \
    golang-any=2:1.22~3~bpo12+1 golang-go=2:1.22~3~bpo12+1 golang-src=2:1.22~3~bpo12+1 \
    gcc-aarch64-linux-gnu gcc-riscv64-linux-gnu
  RUN curl -fsL -o /etc/apt/keyrings/apt-pecke-tt-keyring.asc https://apt.pecke.tt/signing_key.asc \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/apt-pecke-tt-keyring.asc] http://apt.pecke.tt $(. /etc/os-release && echo $VERSION_CODENAME) stable" > /etc/apt/sources.list.d/apt-pecke-tt.list \
    && apt update
  # Build Dependencies
  RUN apt install -y \
    golang-github-dpeckett-archivefs-dev \
    golang-github-dpeckett-telemetry-dev \
    golang-github-dpeckett-uncompr-dev \
    golang-github-google-go-containerregistry-dev \
    golang-github-pierrec-lz4-dev=4.1.18-1~bpo12+1 \
    golang-github-stretchr-testify-dev \
    golang-github-urfave-cli-v2-dev \
    golang-k8s-apimachinery-dev
  RUN mkdir -p /workspace/airgapify
  WORKDIR /workspace/airgapify
  COPY . .
  RUN if [ -n "$(git status --porcelain)" ]; then echo "Please commit your changes."; exit 1; fi
  RUN if [ -z "$(git describe --tags --exact-match 2>/dev/null)" ]; then echo "Current commit is not tagged."; exit 1; fi
  COPY debian/scripts/generate-changelog.sh /usr/local/bin/generate-changelog.sh
  RUN chmod +x /usr/local/bin/generate-changelog.sh
  ENV DEBEMAIL="damian@pecke.tt"
  ENV DEBFULLNAME="Damian Peckett"
  RUN /usr/local/bin/generate-changelog.sh
  RUN VERSION=$(git describe --tags --abbrev=0 | tr -d 'v') \
    && tar -czf ../airgapify_${VERSION}.orig.tar.gz --exclude-vcs .
  ARG GOARCH
  RUN dpkg-buildpackage -d -us -uc --host-arch=${GOARCH}
  SAVE ARTIFACT /workspace/*.deb AS LOCAL dist/
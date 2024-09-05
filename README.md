# airgapify

A little tool that will construct an OCI image archive from a set of Kubernetes manifests.

## Installation

### From APT

Add my apt repository to your system:

*Currently packages are only published for Debian 12 (Bookworm).*

```shell
curl -fsL https://apt.dpeckett.dev/signing_key.asc | sudo tee /etc/apt/keyrings/apt-dpeckett-dev-keyring.asc > /dev/null
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/apt-dpeckett-dev-keyring.asc] http://apt.dpeckett.dev $(. /etc/os-release && echo $VERSION_CODENAME) stable" | sudo tee /etc/apt/sources.list.d/apt-dpeckett-dev.list > /dev/null
```

Then install airgapify:

```shell
sudo apt update
sudo apt install airgapify
```

### GitHub Releases

Download statically linked binaries from the GitHub releases page: 

[Latest Release](https://github.com/dpeckett/airgapify/releases/latest)

## Usage

To create an OCI image archive from a directory containing Kubernetes manifests:

```shell
airgapify -f manifests/ -o images.tar
```

You can then load the image archive into containerd:

```shell
ctr image import images.tar
```

## Configuration

Airgapify will look in the manifests for a Config YAML resource. An example is provided in [examples/config.yaml](examples/config.yaml).

The config resource allows you to specify additional images to include in the archive, and allows configuring image reference extraction for custom resources.

## Telemetry

By default airgapify gathers anonymous crash and usage statistics. This anonymized
data is processed on our servers within the EU and is not shared with third
parties. You can opt out of telemetry by setting the `DO_NOT_TRACK=1`
environment variable.
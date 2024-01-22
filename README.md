# airgapify

A little tool that will construct an OCI image archive from a set of Kubernetes manifests.

## Usage

To create an OCI image archive from a directory containing Kubernetes manifests:

```shell
airgapify -f manifests/ -o images.tar.zst
```

You can then load the image archive into Docker:

```shell
docker load -i images.tar.zst
```

## Configuration

Airgapify will look in the manifests for a Config YAML resource. An example is provided in [examples/config.yaml](examples/config.yaml).

The config resource allows you to specify additional images to include in the archive, and allows configuring image reference extraction for custom resources.
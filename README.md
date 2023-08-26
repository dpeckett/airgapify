# airgapify

A little tool that will construct a Docker image archive from a set of Kubernetes manifests.

## Usage

To create an image archive from a directory containing Kubernetes manifests:

```shell
airgapify -f manifests/ -o images.tar.zst
```

You can then load the image archive into Docker:

```shell
docker load -i images.tar.zst
```
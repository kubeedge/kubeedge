#!/bin/bash

set -e

REPOSITORY=${REPOSITORY:-"kubeedge/build-tools"}
# image tag for build-tools image, including golang version and build-tools version
# If there's some modifications for build-tools.dockerfile other than golang version, the build-tools version should be updated e.g. ke1, ke2.
# If the golang version is updated in build-tools.dockerfile, the build-tools version should be started from ke1.
IMAGE_TAG=${IMAGE_TAG:-"1.17.13-ke1"}
WORK_DIR=$(cd "$(dirname "$0")";pwd)
PUSH_TAG=${1}
ARCHS=(amd64 arm64 arm)

for arch in "${ARCHS[@]}" ; do
  REPOSITORY_ARCH="$REPOSITORY"-"$arch"
  if [ "$arch" = "amd64" ]; then
    REPOSITORY_ARCH="$REPOSITORY"
  fi  
  docker buildx build --build-arg ARCH="$arch"  --platform "$arch" -t "$REPOSITORY_ARCH":"$IMAGE_TAG" -f "$WORK_DIR/build-tools.dockerfile" -o type=docker .
done

if [ "$PUSH_TAG" = 'push' ]; then
  echo "push edgecore image"
  manifestCreateCmd="docker manifest create $REPOSITORY:$IMAGE_TAG"
  for arch in "${ARCHS[@]}" ; do
    REPOSITORY_ARCH="$REPOSITORY"-"$arch"
    if [ "$arch" = "amd64" ]; then
      REPOSITORY_ARCH="$REPOSITORY"
    fi
    docker push "$REPOSITORY_ARCH":"$IMAGE_TAG"
    manifestCreateCmd="$manifestCreateCmd $REPOSITORY_ARCH:$IMAGE_TAG"
  done

  echo "package image manifest and push"
  echo "command: $manifestCreateCmd"
  doCreate="$($manifestCreateCmd)"
  echo "$doCreate"
  docker manifest push --purge "$REPOSITORY":"$IMAGE_TAG"
else
  echo "image save in local"
fi

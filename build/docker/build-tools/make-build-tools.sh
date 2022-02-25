#!/bin/bash

set -e

REPOSITORY=${REPOSITORY:-"kubeedge/build-tools"}
WORK_DIR=$(cd "$(dirname "$0")";pwd)
PUSH_TAG=${1}
ARCHS=(amd64 arm64 arm)

for arch in "${ARCHS[@]}" ; do
  REPOSITORY_ARCH="$REPOSITORY"-"$arch"
  if [ "$arch" = "amd64" ]; then
    REPOSITORY_ARCH="$REPOSITORY"
  fi  
  docker buildx build --build-arg ARCH="$arch"  --platform "$arch" -t "$REPOSITORY_ARCH":latest -f "$WORK_DIR/build-tools.dockerfile" -o type=docker .
done

if [ "$PUSH_TAG" = 'push' ]; then
  echo "push edgecore image"
  manifestCreateCmd="docker manifest create --amend $REPOSITORY:latest"
  for arch in "${ARCHS[@]}" ; do
    REPOSITORY_ARCH="$REPOSITORY"-"$arch"
    if [ "$arch" = "amd64" ]; then
      REPOSITORY_ARCH="$REPOSITORY"
    fi
    docker push "$REPOSITORY_ARCH":latest
    manifestCreateCmd="$manifestCreateCmd $REPOSITORY_ARCH:latest"
  done

  echo "package image manifest and push"
  echo "command: $manifestCreateCmd"
  doCreate="$($manifestCreateCmd)"
  echo "$doCreate"
  docker manifest push "$REPOSITORY":latest
else
  echo "image save in local"
fi

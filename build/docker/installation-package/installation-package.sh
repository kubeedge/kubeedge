#!/bin/bash

set -e

REPOSITORY=${REPOSITORY:-"kubeedge/installation-package"}
RELEASE_VERSION=$(git describe --tags)
pushTag="$1"
WORK_DIR=$(cd "$(dirname "$0")";pwd)
ARCHS=(amd64 arm64 arm)

if ! [ "$(docker version)" ]; then
  echo "docker check failed"
  exit 1
fi

for arch in "${ARCHS[@]}" ; do
  docker buildx build --no-cache --platform "$arch" -t "$REPOSITORY":"$RELEASE_VERSION"-"$arch" -f "$WORK_DIR/installation-package.dockerfile" -o type=docker .
done

if [ "$pushTag" = 'push' ]; then
  echo "push edgecore image"
  manifestCreateCmd="docker manifest create $REPOSITORY:$RELEASE_VERSION"
  for arch in "${ARCHS[@]}" ; do
    docker push "$REPOSITORY":"$RELEASE_VERSION"-"$arch"
    manifestCreateCmd="$manifestCreateCmd $REPOSITORY:$RELEASE_VERSION-$arch"
  done
  echo "package image manifest and push"
  doCreate="$($manifestCreateCmd)"
  echo "$doCreate"
  docker manifest push "$REPOSITORY":"$RELEASE_VERSION"
else
  echo 'image save in local'
fi


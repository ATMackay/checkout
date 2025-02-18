#!/usr/bin/env bash

set -e

commit_hash=$(git rev-parse HEAD)
commit_hash_short=$(git rev-parse --short HEAD)
commit_timestamp=$(git show -s --format="%ci" ${commit_hash})
version_tag=$(git describe --tags)
build_date=$(date -u +'%Y-%m-%d %H:%M:%S')

docker build \
       --build-arg SERVICE=checkout \
       --build-arg GIT_COMMIT="$commit_hash" \
       --build-arg COMMIT_DATE="$commit_timestamp" \
       --build-arg VERSION_TAG="$version_tag" \
       --build-arg BUILD_DATE="$build_date" \
       -t checkout:latest  \
       -t checkout:"$commit_hash_short"  \
       -f Dockerfile .
       
# Remove intermediate Docker layers
docker image prune -f
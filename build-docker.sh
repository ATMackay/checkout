#!/usr/bin/env bash

#!/usr/bin/env bash
set -Eeuo pipefail

IMAGE_NAME=${IMAGE_NAME:-checkout}
SERVICE=${SERVICE:-checkout}

# Git info
commit_hash="$(git rev-parse HEAD)"
commit_hash_short="$(git rev-parse --short=12 HEAD)"
commit_timestamp="$(git show -s --format=%cI "${commit_hash}")"     
build_date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"                        

# Base tag from Git
base_tag="$(git describe --tags --abbrev=0 --match 'v*' 2>/dev/null)"

# Dirty detection: non-empty output -> dirty
if [[ -n "$(git status --porcelain)" ]]; then
  dirty="true"
  version_tag="${base_tag}-dirty-${commit_hash_short}-$(date -u +%Y%m%d%H%M)"
  tags=( "${IMAGE_NAME}:${version_tag}" )  # no :latest for dirty images
else
  dirty="false"
  version_tag="${base_tag}"
  tags=( "${IMAGE_NAME}:${version_tag}" "${IMAGE_NAME}:${commit_hash_short}" "${IMAGE_NAME}:latest" )
fi

echo ">> version_tag=${version_tag}"
echo ">> commit=${commit_hash}"
echo ">> dirty=${dirty}"
echo ">> build_date=${build_date}"

# Build args (propagated into your Makefile's -ldflags via Dockerfile)
build_args="\
  --build-arg SERVICE=${SERVICE} \
  --build-arg GIT_COMMIT=${commit_hash} \
  --build-arg COMMIT_DATE=${commit_timestamp} \
  --build-arg VERSION_TAG=${version_tag} \
  --build-arg BUILD_DATE=${build_date} \
  --build-arg DIRTY=${dirty} \
"

# Primary tag = first in the list
set -- $tags
primary_tag="$1"

# Build image
echo ">> docker build -t $primary_tag $build_args -f Dockerfile ."
docker build -t "$primary_tag" $build_args -f Dockerfile .

# Apply the remaining tags
for t in "${tags[@]:1}"; do
  docker tag "${primary_tag}" "${t}"
done

echo ">> Built tags:"
printf '   %s\n' "${tags[@]}"
       
# Remove intermediate Docker layers
docker image prune -f
#!/usr/bin/env bash
set -Eeuo pipefail

IMAGE_NAME=${IMAGE_NAME:-checkout}

# Git info (used for image tagging only — the binary's commit/date/dirty are
# stamped by the Go toolchain from the copied .git inside the build stage).
commit_hash="$(git rev-parse HEAD)"
commit_hash_short="$(git rev-parse --short=12 HEAD)"

# Base tag from Git
base_tag="$(git describe --tags --abbrev=0 --match 'v*' 2>/dev/null)"

# Dirty detection drives image tagging: dirty builds do not get :latest or a
# clean :<sha> tag.
if [[ -n "$(git status --porcelain)" ]]; then
  version_tag="${base_tag}-dirty-${commit_hash_short}-$(date -u +%Y%m%d%H%M)"
  tags=( "${IMAGE_NAME}:${version_tag}" )
else
  version_tag="${base_tag}"
  tags=( "${IMAGE_NAME}:${version_tag}" "${IMAGE_NAME}:${commit_hash_short}" "${IMAGE_NAME}:latest" )
fi

echo ">> version_tag=${version_tag}"
echo ">> commit=${commit_hash}"

# Only the semver is injected via ldflags; the toolchain stamps the rest.
build_args="--build-arg VERSION=${version_tag}"

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

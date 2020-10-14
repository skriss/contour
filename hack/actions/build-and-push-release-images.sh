#! /usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# This script is intended to be run only for tag builds and will no-op
# if GITHUB_REF is not in the format "refs/tags/<tag-name>".

REF_TYPE=$(echo "$GITHUB_REF" | cut -d / -f 2)
if [[ "$REF_TYPE" != "tags" ]]; then
    echo "REF_TYPE $REF_TYPE is not a tag, exiting."
    exit 0
fi

CURRENT_TAG=$(echo "$GITHUB_REF" | cut -d / -f 3)
if [[ -z "$CURRENT_TAG" ]]; then
    echo "Error getting current tag name from GITHUB_REF $GITHUB_REF."
    exit 1
fi

git fetch --tags

HIGHEST_SEMVER_TAG=""
for t in $(git tag -l --sort=-v:refname); do
    # Skip pre-release tags
    if [[ "$t" == *"beta"* || "$t" == *"alpha"* || "$t" == *"rc"* ]]; then
        continue
    fi
    HIGHEST_SEMVER_TAG="$t"
    break
done

echo "CURRENT_TAG: $CURRENT_TAG"
echo "HIGHEST_SEMVER_TAG: $HIGHEST_SEMVER_TAG"

TAG_LATEST="false"
if [[ "$CURRENT_TAG" != "$HIGHEST_SEMVER_TAG" ]]; then
    echo "Current tag is not the highest semver tag, image will not be tagged as 'latest'."
else
    echo "Current tag is the highest semver tag, image will also be tagged as 'latest'."
    TAG_LATEST="true"
fi

make push TAG_LATEST="$TAG_LATEST"

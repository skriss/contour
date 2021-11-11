#! /usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

readonly HERE=$(cd "$(dirname "$0")" && pwd)
readonly REPO=$(cd "${HERE}/../.." && pwd)

ytt -f ${REPO}/examples/contour/ -f ${HERE}/ytt/local-contour-deployment.yaml --data-value xdsAddress=${XDS_ADDRESS} | kubectl apply -f -

go run ./cmd/contour serve --xds-address=0.0.0.0 --config-path ${HERE}/contour-config.yml --insecure --disable-leader-election

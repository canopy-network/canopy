#!/usr/bin/env bash
#
# Build the canopy-core docker image from the current checkout, import it
# directly into a k3s node's containerd store (no registry involved -- k3s
# supports `ctr images import` for exactly this), and optionally patch a
# StatefulSet in the cluster to roll out the new tag.
#
# Usage:
#   scripts/deploy-node-image.sh [--statefulset NAME] [--image NAME] [--skip-patch] [--no-wait]
#
# Flags (each overrides the matching env var below):
#   -s, --statefulset NAME   StatefulSet to patch
#   -i, --image NAME         docker image repo name
#
# Configuration (env vars, all optional):
#   IMAGE_NAME    docker image repo name              (default: canopy-lp-cap)
#   TAG           image tag                           (default: current git short sha)
#   DOCKERFILE    Dockerfile to build                 (default: .docker/Dockerfile)
#   NODE          SSH host of the target k3s node     (default: val-b)
#   KUBE_HOST     SSH host with kubectl access         (default: staging)
#   NAMESPACE     kubernetes namespace                (default: staging)
#   STATEFULSET   StatefulSet to patch                (default: localnet-2-localnet)
#   CONTAINER     container name inside the pod       (default: canopy)
#   WAIT_TIMEOUT  seconds to wait for pod readiness    (default: 120)
#
# Example:
#   NODE=val-a scripts/deploy-node-image.sh --statefulset localnet-1-localnet
#   scripts/deploy-node-image.sh -s localnet-1-localnet -i canopy-lp-cap

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

IMAGE_NAME="${IMAGE_NAME:-canopy-lp-cap}"
TAG="${TAG:-$(git rev-parse --short HEAD)}"
DOCKERFILE="${DOCKERFILE:-.docker/Dockerfile}"
NODE="${NODE:-val-b}"
KUBE_HOST="${KUBE_HOST:-staging}"
NAMESPACE="${NAMESPACE:-staging}"
STATEFULSET="${STATEFULSET:-localnet-2-localnet}"
CONTAINER="${CONTAINER:-canopy}"
WAIT_TIMEOUT="${WAIT_TIMEOUT:-120}"

SKIP_PATCH=0
NO_WAIT=0
while [ $# -gt 0 ]; do
  case "$1" in
    --skip-patch) SKIP_PATCH=1; shift ;;
    --no-wait) NO_WAIT=1; shift ;;
    -s|--statefulset)
      STATEFULSET="$2"
      shift 2
      ;;
    -i|--image)
      IMAGE_NAME="$2"
      shift 2
      ;;
    -h|--help)
      grep '^#' "${BASH_SOURCE[0]}" | tail -n +2 | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

IMAGE_REF="${IMAGE_NAME}:${TAG}"
TAR_NAME="${IMAGE_NAME}-${TAG}.tar"
LOCAL_TAR="$(mktemp -t "${TAR_NAME}.XXXXXX")"
REMOTE_TAR="/tmp/${TAR_NAME}"

cleanup() {
  rm -f "$LOCAL_TAR"
}
trap cleanup EXIT

echo "==> building ${IMAGE_REF} from ${DOCKERFILE}"
docker build -f "$DOCKERFILE" -t "$IMAGE_REF" .

echo "==> saving image to ${LOCAL_TAR}"
docker save "$IMAGE_REF" -o "$LOCAL_TAR"

echo "==> copying image to ${NODE}:${REMOTE_TAR}"
scp "$LOCAL_TAR" "${NODE}:${REMOTE_TAR}"

echo "==> importing image into ${NODE}'s containerd via k3s ctr"
ssh "$NODE" "k3s ctr images import '${REMOTE_TAR}' && rm -f '${REMOTE_TAR}'"

if [ "$SKIP_PATCH" -eq 1 ]; then
  echo "==> --skip-patch set, image imported but no deployment was changed"
  exit 0
fi

echo "==> patching statefulset/${STATEFULSET} (container ${CONTAINER}) to ${IMAGE_REF} in namespace ${NAMESPACE}"
ssh "$KUBE_HOST" "kubectl set image statefulset/${STATEFULSET} ${CONTAINER}=${IMAGE_REF} -n ${NAMESPACE}"

if [ "$NO_WAIT" -eq 1 ]; then
  echo "==> --no-wait set, not waiting for pod readiness"
  exit 0
fi

echo "==> waiting up to ${WAIT_TIMEOUT}s for ${STATEFULSET}-0's pod to become ready"
ssh "$KUBE_HOST" "kubectl rollout status statefulset/${STATEFULSET} -n ${NAMESPACE} --timeout=${WAIT_TIMEOUT}s"

echo "==> done: ${STATEFULSET} is running ${IMAGE_REF}"

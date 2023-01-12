#!/usr/bin/env bash
set -euo pipefail

failure() {
  echo "Error happened"
  exit 1
}
trap failure ERR

TAG=${TAG:-staging}
echo "Using tag $TAG"

# Explicit loging
export AWS_DEFAULT_REGION=${AWS_DEFAULT_REGION:-"us-east-1"}
# forcefully set REGISTRY
export REGISTRY="${ACCOUNT}.dkr.ecr.us-east-1.amazonaws.com"
aws ecr get-login-password --region ${AWS_DEFAULT_REGION} | docker login --username AWS --password-stdin ${REGISTRY}

IMAGE=${IMAGE:-${ACCOUNT}.dkr.ecr.us-east-1.amazonaws.com/macaroon_vault}

docker pull $IMAGE:$TAG
docker tag $IMAGE:$TAG ghcr.io/bolt-observer/macaroon_vault:latest
docker push ghcr.io/bolt-observer/macaroon_vault:latest

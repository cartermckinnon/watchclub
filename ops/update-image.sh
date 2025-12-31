#!/usr/bin/env bash

if [ $# -ne 6 ]; then
  echo >&2 "usage: NAMESPACE RESOURCE_TYPE RESOURCE_NAME CONTAINER_NAME IMAGE_NAME IMAGE_TAG"
  exit 1
fi

NAMESPACE=$1
RESOURCE_TYPE=$2
RESOURCE_NAME=$3
CONTAINER_NAME=$4
IMAGE_NAME=$5
IMAGE_TAG=$6

echo "Checking ${RESOURCE_TYPE}/${RESOURCE_NAME}..."

CURRENT_IMAGE=$(kubectl get $RESOURCE_TYPE $RESOURCE_NAME -n $NAMESPACE -o jsonpath="{.spec.template.spec.containers[?(@.name=='$CONTAINER_NAME')].image}")
echo "Current: $CURRENT_IMAGE"

echo "Querying registry for ${IMAGE_NAME}:${IMAGE_TAG}..."
NEW_DIGEST=$(crane digest "${IMAGE_NAME}:${IMAGE_TAG}" 2>&1)

if [ $? -ne 0 ] || [ -z "$NEW_DIGEST" ]; then
  echo "WARNING: Could not query registry"
  echo "Output: $NEW_DIGEST"
  exit 1
fi

NEW_IMAGE="${IMAGE_NAME}@${NEW_DIGEST}"
echo "Available:  $NEW_IMAGE"

if [ "$CURRENT_IMAGE" = "$NEW_IMAGE" ]; then
  echo "Already up to date"
  exit 0
fi

echo "New image detected, updating..."
kubectl set image $RESOURCE_TYPE/$RESOURCE_NAME -n $NAMESPACE \
  ${CONTAINER_NAME}="${NEW_IMAGE}"

if [ $? -eq 0 ]; then
  echo "Updated successfully"
else
  echo "Update failed"
  exit 1
fi

#!/bin/bash

set -e

# Usage: ./scripts/docker-build-push.sh --repo <repository-path> [--tag <image-tag>]

# Find project root (parent of script directory) without cd
SCRIPT=$(readlink -f "$0")
SCRIPTPATH=$(dirname "$SCRIPT")
PROJECT_ROOT="$SCRIPTPATH/.."

TAG="latest"

# Parse arguments
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    --repo)
      REPO="$2"
      shift # past argument
      shift # past value
      ;;
    --tag)
      TAG="$2"
      shift # past argument
      shift # past value
      ;;
    *)
      echo "Unknown option: $1"
      echo "Usage: $0 --repo <repository-path> [--tag <image-tag>]"
      exit 1
      ;;
  esac
done

if [ -z "$REPO" ]; then
  echo "Error: --repo <repository-path> is required."
  echo "Usage: $0 --repo <repository-path> [--tag <image-tag>]"
  exit 1
fi

# Build and publish the image using ko
KO_CMD="ko publish --tags=$TAG --repository=$REPO $PROJECT_ROOT/main.go"
echo "Running: $KO_CMD"
$KO_CMD

echo "Image built and published to $REPO with tag $TAG using ko." 
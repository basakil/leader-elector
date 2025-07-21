#!/bin/bash

set -e

print_help() {
  cat <<EOF
Build and publish a container image using ko.

Usage:
  $0 --repo <repository-path> [--tag <image-tag>]

Mandatory parameters:
  --repo <repository-path>   The container repository path (e.g., gcr.io/my-project/leader-elector)

Optional parameters:
  --tag <image-tag>          The image tag to use (default: latest)
  -h, --help                 Show this help message and exit

Examples:
  $0 --repo gcr.io/my-project/leader-elector
  $0 --repo docker.io/myuser/leader-elector --tag v1.2.3
EOF
}

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
    -h|--help)
      print_help
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      print_help
      exit 1
      ;;
  esac
done

if [ -z "$REPO" ]; then
  echo "Error: --repo <repository-path> is required."
  print_help
  exit 1
fi

# Build and publish the image using ko
KO_CMD="ko publish --tags=$TAG --repository=$REPO $PROJECT_ROOT/main.go"
echo "Running: $KO_CMD"
$KO_CMD

echo "Image built and published to $REPO with tag $TAG using ko." 
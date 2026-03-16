#!/bin/bash

APP_NAME="nvim-ai-engine"
BUILD_DIR="dist"
MAIN_PKG="./cmd/engine"

if [ "$CI" = "true" ]; then
  VERSION=$(git describe --tags --always)
  echo "=> Running in CI mode for version: ${VERSION}"
else
  echo "=> Fetching latest tags from remote..."
  git fetch --tags

  LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
  echo "=> Latest version found: ${LATEST_TAG}"

  if [[ $LATEST_TAG =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    MAJOR=${BASH_REMATCH[1]}
    MINOR=${BASH_REMATCH[2]}
    PATCH=${BASH_REMATCH[3]}
    NEXT_TAG="v$MAJOR.$MINOR.$((PATCH + 1))"
  else
    NEXT_TAG="v0.0.1"
  fi

  read -p "=> Enter version to release [$NEXT_TAG]: " USER_VERSION
  VERSION=${USER_VERSION:-$NEXT_TAG}
fi

PLATFORMS=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
)

echo "=> Starting the release process for ${VERSION}..."
mkdir -p "${BUILD_DIR}"

for PLATFORM in "${PLATFORMS[@]}"; do
  OS=${PLATFORM%/*}
  ARCH=${PLATFORM#*/}

  OUTPUT_NAME="${APP_NAME}-${OS}-${ARCH}"
  BINARY_PATH="${BUILD_DIR}/${OUTPUT_NAME}"

  echo "=> Building for ${OS}/${ARCH}..."

  GOOS=${OS} GOARCH=${ARCH} go build \
    -ldflags="-s -w -X 'main.Version=${VERSION}'" \
    -trimpath \
    -o "${BINARY_PATH}" \
    "${MAIN_PKG}"

  if command -v upx >/dev/null; then
    echo "   -> Compressing with UPX..."
    upx -q "${BINARY_PATH}" >/dev/null
  fi

  TAR_FILE="${BUILD_DIR}/${APP_NAME}-${VERSION}-${OS}-${ARCH}.tar.gz"
  tar -czf "${TAR_FILE}" -C "${BUILD_DIR}" "${OUTPUT_NAME}"

  rm "${BINARY_PATH}"

  echo "   -> Created: ${TAR_FILE}"
done

if [ "$CI" != "true" ]; then
  echo "=> Release build successful!"
  read -p "=> Do you want to tag this release as ${VERSION} and push to origin? (y/n): " CONFIRM
  if [[ $CONFIRM == [yY] ]]; then
    echo "=> Tagging ${VERSION}..."
    git tag -a "${VERSION}" -m "Release ${VERSION}"
    git push origin "${VERSION}"
    echo "=> Tag ${VERSION} pushed successfully!"
  else
    echo "=> Skipping Git tagging."
  fi
fi

echo "=> All releases are prepared in the /${BUILD_DIR} directory"

#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -e

REGISTRY_RS_URL="registry-rs.edgeorchestration.intel.com"
S3_BUCKET=$1

ALL_PROFILES=$(find ../../../os-profiles/*.yaml)


# Check if the URL exists with a HEADER request.
check_url() {
  local url=$1
  resp=$(curl -sLI -o /dev/null -w "%{http_code}" "$url")
  if [ "$resp" != "200" ]; then
    echo "$url does not exist (http_code=$resp)"
    exit 1
  fi
}

# Check if the S3 resource exists.
check_s3_resource() {
  local res=$1
  if ! aws s3 ls "s3://$S3_BUCKET/$res"; then
    echo "$res does not exist"
    exit 1
  fi
}

# Check if the OCI artifact exists.
check_oci_artifact() {
  local artifact=$1
  if ! oras manifest fetch "$REGISTRY_RS_URL/$artifact" > /dev/null; then
    echo "$artifact does not exist"
    exit 1
  fi
}

for file in ${ALL_PROFILES}; do
  echo "Checking $file"
  os_name=$(yq e '.spec.name' "${file}")
  os_image_url=$(yq e '.spec.osImageUrl' "${file}");
  os_package_manifest_url=$(yq e '.spec.osPackageManifestURL' "${file}")
  os_image_type=$(yq e '.spec.type' "${file}")
  echo "    name                              = $os_name"
  echo "    image URL                         = $os_image_url"
  echo "    OS Package manifest URL           = $os_package_manifest_url"
  echo "    OS Type                           = $os_image_url"
  case "$os_image_type" in
    "OS_TYPE_MUTABLE")
      # case for ubuntu, pull from upstream
      if [ -n "$os_image_url" ]; then
        check_url "$os_image_url"
      fi
      ;;
    "OS_TYPE_IMMUTABLE")
      # case for microvisor, pull from RS
      check_s3_resource "$os_image_url"
      ;;
  esac
  if [ -n "$os_package_manifest_url" ]; then
    check_s3_resource "$os_package_manifest_url"
  fi
  platformBundleArtifacts=$(yq e '.spec.platformBundle | to_entries | .[].value' "${file}")
  for artifact in ${platformBundleArtifacts}; do
    echo "    Checking platform bundle artifact = $artifact"
    check_oci_artifact "$artifact"
  done
  echo "-----------------------------------"
done

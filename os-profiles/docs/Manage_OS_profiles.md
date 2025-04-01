# Managing OS profiles

OS profiles defines a flavor of an Operating System (OS) that is used for Edge Node provisioning.
All available OS profiles are stored as YAML manifest files under [os-profiles](../../os-profiles).

Each YAML manifest file defines a single OS profile and must set pre-defined OS profile's parameters like
OS image URL, OS image version, SHA256 of OS image, etc. The meaning and usage of all fields are documented in the
[OS profile template file](../template/profile-template.yaml).

## Updating existing OS profile

There are already several OS profiles defined in this repository and available for users of Edge Manageability
Framework.
Over time OS profiles may need to be updated (e.g., to upgrade to a newer OS image version). Then, the YAML files
need to be modified to update the OS profile and merged to the mainline.

A general guidelines to follow when updating an OS profile:

- when changing the OS image always remember to update SHA256 as well.
- when changing the OS image make sure that the OS image version is in line with the image. OS image version
  should point to the version that is retrievable from `/etc/os-release` (or any other standard OS location) from a
  running OS.
- when changing the OS image version make should it's also reflected in the `name` field as it's exposed as
  humand-readable OS profile name.
  The `name` field should uniquely identify the OS profile.

### Example: Updating OS image version of Ubuntu 22.04 profile

An example of Ubuntu 22.04 profile is defined in the
[ubuntu-22.04-lts-generic.yaml](../ubuntu-22.04-lts-generic.yaml) file.

For Ubuntu we use stable releases that are published under [cloud-images.ubuntu.com/releases](https://cloud-images.ubuntu.com/releases).
Let's assume we want to update to version released in [https://cloud-images.ubuntu.com/releases/22.04/release-20250108/](https://cloud-images.ubuntu.com/releases/22.04/release-20250108/).
Perform the following steps:

1. Enter the link above and look for the .img file URL. In this case it's [https://cloud-images.ubuntu.com/releases/22.04/release-20250108/ubuntu-22.04-server-cloudimg-amd64.img](https://cloud-images.ubuntu.com/releases/22.04/release-20250108/ubuntu-22.04-server-cloudimg-amd64.img).
   Copy the link and paste as `os_image_url` to the OS profile manifest.
2. Find the SHA256 checksum of the image. It's usually provided in the same directory as the image file. In this case
   it's [https://cloud-images.ubuntu.com/releases/22.04/release-20250108/SHA256SUMS](https://cloud-images.ubuntu.com/releases/22.04/release-20250108/SHA256SUMS).
   Open the file and find the checksum for the image file. Copy the checksum and paste as `os_image_sha256` to the
   OS profile manifest.
3. Set the `os_image_version`. If you use Ubuntu LTS you can simply set `22.04` as it's the version of the image.
   Otherwise, to point to the release build you have to find the version in the image.
   You can do it by downloading the image and running `cat /etc/os-release` on it. The `VERSION_ID` field should
   contain the version.
4. Update the `name` field to reflect the new version. For Ubuntu LTS it can be `Ubuntu 22.04 LTS`, but if you want
   to explicitly point to the release build you should specify the full version (e.g., `Ubuntu 22.04.5`).

After updating the fields, the manifest should look like this:

```yaml
name: Ubuntu 22.04.5 LTS
os_image_url: https://cloud-images.ubuntu.com/releases/22.04/release-20250108/ubuntu-22.04-server-cloudimg-amd64.img  
os_image_sha256: 610e1d37fe06fa23db2ff34d1c03711e55d73b7c3e4482503f50c4d2f9b9d06d
os_image_version: 22.04.5
```

### Example: Updating OS image version of Edge Microvisor Toolkit non-RT profile

An example of Edge Microvisor Toolkit non-RT profile is defined in the
[microvisor-non-rt.yaml](../microvisor-nonrt.yaml) file.

Edge Microvisor Toolkit is an immutable OS developed and distributed by Intel. Let's assume we want to
update Edge Microvisor Toolkit version to `3.0.20250105.2206`.
The Edge Microvisor Toolkit image is stored on the Release Service (RS) file server under the following subpath
`files-edge-orch/repository/microvisor/non_rt/edge-readonly-dev-3.0.20250324.1008.raw.gz`.

To update the Edge Microvisor Toolkit profile perform the following steps:

1. Set `os_image_url` as the RS file server subpath. In this case it's
   `files-edge-orch/repository/microvisor/non_rt/edge-readonly-dev-3.0.20250324.1008.raw.gz`.
   The orchestrator will automatically expand the URL to the full URL based on the cluster configuration
   (the RS endpoint is cluster-specific).
2. Set `os_image_version` to `3.0.20250324.1008`.
3. Update the `name` field to reflect the new version. It can be `Edge Microvisor Toolkit Microvisor 3.0.20250324`
   (you can omit the last part that identifies the build ID).
4. Update the `os_image_sha256` field. The SHA256 checksum can be obtained by downloading the image and running
   `sha256sum` on it or by downloading `.sha256sum` file that is associated with the Edge Microvisor Toolkit image.

After updating the fields, the manifest should look like this:

```yaml
 name: Edge Microvisor Toolkit 3.0.20250324
 osImageUrl: files-edge-orch/repository/microvisor/non_rt/edge-readonly-dev-3.0.20250324.1008.raw.gz
 osImageVersion: 3.0.20250324.1008
 osImageSha256: 89d691eded21e158e94cf52235106d8eb6c17f81f37b1a79c70514776744bc74
```

## Adding new OS profile

If you want Edge Manageability Framework to support a new OS profile (e.g., for new version of Ubuntu like 24.04, etc.),
you need to create a new YAML manifest file in the [top directory](../).

It's important to create a profile with a unique `profile_name` (cannot be the same as existing ones) and the file name
should reflect the profile name (e.g., `ubuntu-24.04.yaml`).
Remember that the new OS profile should be based on the
[OS profile template file](../template/profile-template.yaml) and should define all required fields.

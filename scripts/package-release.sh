#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 4 ]]; then
    echo "usage: $0 <amd64|arm64> <panel-binary> <runtime-zip> <output-directory>" >&2
    exit 2
fi

target_arch="$1"
panel_binary="$(readlink -f "$2")"
runtime_zip="$(readlink -f "$3")"
output_dir="$(mkdir -p "$4" && cd "$4" && pwd)"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

case "${target_arch}" in
    amd64)
        runtime_name="Xray-linux-64.zip"
        machine_pattern="x86-64|x86_64"
        ;;
    arm64)
        runtime_name="Xray-linux-arm64.zip"
        machine_pattern="ARM aarch64|AArch64"
        ;;
    *) echo "unsupported target architecture: ${target_arch}" >&2; exit 2 ;;
esac

[[ -x "${panel_binary}" ]] || { echo "panel binary is not executable" >&2; exit 1; }
[[ -f "${runtime_zip}" ]] || { echo "runtime ZIP not found" >&2; exit 1; }
file "${panel_binary}" | grep -Eq "${machine_pattern}" || {
    echo "panel binary architecture does not match ${target_arch}" >&2
    exit 1
}
unzip -t "${runtime_zip}" >/dev/null
unzip -l "${runtime_zip}" | grep -Eq '[[:space:]]xray$'
unzip -l "${runtime_zip}" | grep -Eq '[[:space:]]geoip.dat$'
unzip -l "${runtime_zip}" | grep -Eq '[[:space:]]geosite.dat$'

stage="$(mktemp -d)"
trap 'rm -rf "${stage}"' EXIT
mkdir -p "${stage}/x-ui/releases"
rsync -a \
    --exclude='/.git' \
    --exclude='/x-ui' \
    --exclude='/releases/*.zip' \
    --exclude='/releases/*.tar.gz' \
    "${repo_root}/" "${stage}/x-ui/"
install -m 0755 "${panel_binary}" "${stage}/x-ui/x-ui"
install -m 0644 "${runtime_zip}" "${stage}/x-ui/releases/${runtime_name}"
runtime_sha="$(sha256sum "${runtime_zip}" | awk '{print $1}')"
{
    echo "98e1cfe7b8a85d833edcd5101530f2d67609505d832eac15ac2a236f3374bbbe  Xray-linux-64.zip"
    if [[ "${target_arch}" == "arm64" ]]; then
        echo "${runtime_sha}  Xray-linux-arm64.zip"
    fi
} >"${stage}/x-ui/releases/SHA256SUMS"

source_date_epoch="$(git -C "${repo_root}" show -s --format=%ct HEAD)"
package="${output_dir}/x-ui-linux-${target_arch}.tar.gz"
tar --sort=name --mtime="@${source_date_epoch}" --owner=0 --group=0 --numeric-owner \
    -C "${stage}" -cf - x-ui | gzip -n -9 >"${package}"

sha256sum "${package}"

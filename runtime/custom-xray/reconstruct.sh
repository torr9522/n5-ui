#!/usr/bin/env bash
set -euo pipefail

UPSTREAM_REPOSITORY="https://github.com/XTLS/Xray-core.git"
UPSTREAM_COMMIT="228f1e13aa22739b0d6b9adbdb2b600f1e2018e1"
RECONSTRUCTED_TREE="94bed1effd3c690b45b1b2978566d710b08723d6"
RECONSTRUCTED_COMMIT="6ed7a248b8e21940f365ceeffcc4e53bb42eb1f3"
RECONSTRUCTED_TIMESTAMP="1791102074 +0000"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
patch_file="${script_dir}/patches/N5-CUSTOM-XRAY-26.5.3.patch"

if [[ $# -ne 1 ]]; then
    echo "usage: $0 <empty-output-directory>" >&2
    exit 2
fi

output="$1"
if [[ -e "${output}" ]]; then
    echo "output already exists: ${output}" >&2
    exit 1
fi

git clone --no-checkout "${UPSTREAM_REPOSITORY}" "${output}"
git -C "${output}" checkout --detach "${UPSTREAM_COMMIT}"
git -C "${output}" apply --check "${patch_file}"
git -C "${output}" apply "${patch_file}"

temporary_index="$(mktemp)"
trap 'rm -f "${temporary_index}"' EXIT
rm -f "${temporary_index}"
GIT_INDEX_FILE="${temporary_index}" git -C "${output}" read-tree HEAD
GIT_INDEX_FILE="${temporary_index}" git -C "${output}" add -A
actual_tree="$(GIT_INDEX_FILE="${temporary_index}" git -C "${output}" write-tree)"

[[ "${actual_tree}" == "${RECONSTRUCTED_TREE}" ]] || {
    echo "reconstructed tree mismatch: ${actual_tree}" >&2
    exit 1
}

git -C "${output}" add -A
GIT_AUTHOR_NAME="N5 ARM64 Reconstruction" \
GIT_AUTHOR_EMAIL="n5-arm64-reconstruction@localhost" \
GIT_AUTHOR_DATE="${RECONSTRUCTED_TIMESTAMP}" \
GIT_COMMITTER_NAME="N5 ARM64 Reconstruction" \
GIT_COMMITTER_EMAIL="n5-arm64-reconstruction@localhost" \
GIT_COMMITTER_DATE="${RECONSTRUCTED_TIMESTAMP}" \
    git -C "${output}" commit -m "n5: reconstruct custom Xray 26.5.3 delta" >/dev/null

actual_commit="$(git -C "${output}" rev-parse HEAD)"
[[ "${actual_commit}" == "${RECONSTRUCTED_COMMIT}" ]] || {
    echo "deterministic reconstructed commit mismatch: ${actual_commit}" >&2
    exit 1
}

echo "reconstructed Custom Xray tree: ${actual_tree}"
echo "reconstructed provenance commit: ${actual_commit}"

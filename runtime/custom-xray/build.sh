#!/usr/bin/env bash
set -euo pipefail

UPSTREAM_COMMIT="228f1e13aa22739b0d6b9adbdb2b600f1e2018e1"
RECONSTRUCTED_TREE="94bed1effd3c690b45b1b2978566d710b08723d6"
REQUIRED_GO_VERSION="go1.26.0"

usage() {
    echo "usage: $0 <source-dir> <amd64|arm64> <output> [go-binary]" >&2
    exit 2
}

[[ $# -ge 3 && $# -le 4 ]] || usage

source_dir="$(cd "$1" && pwd)"
target_arch="$2"
output="$3"
go_binary="${4:-go}"

case "${target_arch}" in
    amd64|arm64) ;;
    *)
        echo "unsupported target architecture: ${target_arch}" >&2
        exit 2
        ;;
esac

[[ -f "${source_dir}/go.mod" && -d "${source_dir}/main" ]] || {
    echo "not an Xray source tree: ${source_dir}" >&2
    exit 1
}

actual_go_version="$(${go_binary} version | awk '{print $3}')"
[[ "${actual_go_version}" == "${REQUIRED_GO_VERSION}" ]] || {
    echo "Go ${REQUIRED_GO_VERSION#go} is required; found ${actual_go_version}" >&2
    exit 1
}

if [[ -e "${source_dir}/.git" ]]; then
    temporary_index="$(mktemp)"
    trap 'rm -f "${temporary_index}"' EXIT
    rm -f "${temporary_index}"
    GIT_INDEX_FILE="${temporary_index}" git -C "${source_dir}" read-tree HEAD
    GIT_INDEX_FILE="${temporary_index}" git -C "${source_dir}" add -A
    actual_tree="$(GIT_INDEX_FILE="${temporary_index}" git -C "${source_dir}" write-tree)"
elif git -C "${source_dir}" rev-parse --show-toplevel >/dev/null 2>&1; then
    repo_root="$(git -C "${source_dir}" rev-parse --show-toplevel)"
    relative_source="${source_dir#${repo_root}/}"
    actual_tree="$(git -C "${repo_root}" rev-parse "HEAD:${relative_source}")"
else
    echo "Custom Xray source must be tracked by Git so its tree can be verified" >&2
    exit 1
fi

[[ "${actual_tree}" == "${RECONSTRUCTED_TREE}" ]] || {
    echo "Custom Xray source tree mismatch: ${actual_tree}" >&2
    echo "expected reconstructed tree: ${RECONSTRUCTED_TREE}" >&2
    exit 1
}

mkdir -p "$(dirname "${output}")"
(
    cd "${source_dir}"
    CGO_ENABLED=1 GOOS=linux GOARCH="${target_arch}" \
        "${go_binary}" build -buildvcs=true -o "${output}" ./main
)

echo "built N5 Custom Xray from upstream ${UPSTREAM_COMMIT}"
echo "source tree: ${RECONSTRUCTED_TREE}"
echo "output: ${output}"

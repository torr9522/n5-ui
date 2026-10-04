#!/usr/bin/env bash
set -euo pipefail

REQUIRED_GO_VERSION="go1.26.0"

if [[ $# -lt 2 || $# -gt 3 ]]; then
    echo "usage: $0 <amd64|arm64> <output> [go-binary]" >&2
    exit 2
fi

target_arch="$1"
output="$2"
go_binary="${3:-go}"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

case "${target_arch}" in
    amd64|arm64) ;;
    *) echo "unsupported target architecture: ${target_arch}" >&2; exit 2 ;;
esac

actual_go_version="$(${go_binary} version | awk '{print $3}')"
[[ "${actual_go_version}" == "${REQUIRED_GO_VERSION}" ]] || {
    echo "Go ${REQUIRED_GO_VERSION#go} is required; found ${actual_go_version}" >&2
    exit 1
}

[[ -z "$(git -C "${repo_root}" status --porcelain --untracked-files=no)" ]] || {
    echo "tracked panel source must be clean before a formal build" >&2
    exit 1
}

mkdir -p "$(dirname "${output}")"
(
    cd "${repo_root}"
    CGO_ENABLED=1 GOOS=linux GOARCH="${target_arch}" \
        "${go_binary}" build -buildvcs=true -o "${output}" .
)

echo "panel source: $(git -C "${repo_root}" rev-parse HEAD)"
echo "output: ${output}"

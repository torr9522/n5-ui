#!/usr/bin/env bash
set -euo pipefail

GOLDEN_GEOIP_SHA256="85fff3c5811f07fd163e6fc83c8891c498003159dcb7a488856fe48538877184"
GOLDEN_GEOSITE_SHA256="61a399fcd21c300cf8150299539d1333aa006ac656dc0c0d2d3cfc95dd74a4ab"

if [[ $# -ne 4 ]]; then
    echo "usage: $0 <xray-binary> <geoip.dat> <geosite.dat> <output.zip>" >&2
    exit 2
fi

xray_binary="$1"
geoip="$2"
geosite="$3"
output_dir="$(mkdir -p "$(dirname "$4")" && cd "$(dirname "$4")" && pwd)"
output="${output_dir}/$(basename "$4")"

[[ -x "${xray_binary}" ]] || { echo "xray binary is not executable: ${xray_binary}" >&2; exit 1; }
[[ "$(sha256sum "${geoip}" | awk '{print $1}')" == "${GOLDEN_GEOIP_SHA256}" ]] || {
    echo "geoip.dat does not match the N5 Golden asset" >&2
    exit 1
}
[[ "$(sha256sum "${geosite}" | awk '{print $1}')" == "${GOLDEN_GEOSITE_SHA256}" ]] || {
    echo "geosite.dat does not match the N5 Golden asset" >&2
    exit 1
}

stage="$(mktemp -d)"
trap 'rm -rf "${stage}"' EXIT
install -m 0755 "${xray_binary}" "${stage}/xray"
install -m 0755 "${geoip}" "${stage}/geoip.dat"
install -m 0755 "${geosite}" "${stage}/geosite.dat"
source_date_epoch="${SOURCE_DATE_EPOCH:-1777809193}"
touch -d "@${source_date_epoch}" "${stage}/xray" "${stage}/geoip.dat" "${stage}/geosite.dat"
rm -f "${output}"
(
    cd "${stage}"
    zip -X -9 "${output}" xray geoip.dat geosite.dat
)

sha256sum "${output}"

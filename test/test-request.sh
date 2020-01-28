#!/bin/sh

wd="$(dirname "$0")"
: "${crt:=orchestrator.crt}"
: "${ca_crt:=server.crt}"
: "${key:=orchestrator.key}"
: "${gen_crt_sh:=$wd/gen-certs.sh}"
: "${oc2_url:=https://127.0.0.1:1512/oc2}"
: "${request_id:=$(uuid || date +%s)}"

if which jq >/dev/null;then
  _jq() { jq . "$@"; }
else
  _jq() { cat "$@"; }
fi

test -e "$1" || {
  echo "Usage: ${0##*/} <json_file>" >&2
  exit 1
}

r() { (set -x; "$@"); }

set -e
test -e "$crt" || r "$gen_crt_sh"

echo "--> COMMAND"
_jq "$1"
resp_file="$(mktemp resp-${request_id}-XXXXXX.json)"
hdr_file="${resp_file%.json}.hdr"
echo "<-- RESPONSE in $resp_file"
r curl -D "$hdr_file" -o "$resp_file" \
  --cacert "$ca_crt" --cert "$crt" --key "$key" \
  -H "X-Request-ID: $request_id" \
  --data-binary "@$1" \
  "$oc2_url"

cat "$hdr_file"
_jq "$resp_file"

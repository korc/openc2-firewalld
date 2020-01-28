#!/bin/sh

test -n "${server_ip+set}" || server_ip="127.0.0.1"
test -n "${server_name+set}" || server_name="localhost"

: "${client:=client}"
: "${self:=OpenC2-FirewallD}"

r() { (set -x; "$@"); }

set -e

test -x "$(which certtool)" || {
  echo "ERROR: Need certtool from gnutls-bin package" >&2
  exit 1
}

echo "env: server_ip='$server_ip' server_name='$server_name' client='$client' self='$self'"

for key in server.key "${client}.key" ca.key orchestrator.key;do
  test -s "$key" || r certtool -p --outfile "$key"
done

test -s "ca.crt" || {
  test -e ca.tmpl || cat >"ca.tmpl" <<EOF
cn=$self HTTPS Client CA
ca
EOF
  r certtool -s --load-privkey ca.key --template ca.tmpl --outfile ca.crt
}

test -s "server.crt" || {
  test -e server.tmpl || cat >server.tmpl <<EOF
cn=$self HTTPS Server
${server_ip:+ip_address = "$server_ip"}
${server_name:+dns_name = "$server_name"}
tls_www_server
ca
EOF
  r certtool -s --load-privkey server.key --template server.tmpl --outfile server.crt
}

test -s "${client}.crt" || {
  test -e "${client}.tmpl" || cat >"${client}.tmpl" <<EOF
cn=$self ${client}
tls_www_client
EOF
  r certtool -c --load-ca-certificate ca.crt --load-ca-privkey ca.key --load-privkey "${client}.key" --template "${client}.tmpl" --outfile "${client}.crt"
}

test -s "orchestrator.crt" || {
  test -e orchestrator.tmpl || cat >orchestrator.tmpl <<EOF
cn=$self Orchestrator
tls_www_client
EOF
  r certtool -c --load-ca-certificate ca.crt --load-ca-privkey ca.key --load-privkey orchestrator.key --template orchestrator.tmpl --outfile orchestrator.crt
}

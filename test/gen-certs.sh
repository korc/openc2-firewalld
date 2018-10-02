#!/bin/sh

set -e
for key in server.key client.key ca.key;do
  test -s "$key" || (set -x; certtool -p --outfile "$key")
done

test -s "ca.crt" || {
  test -e ca.tmpl || cat >"ca.tmpl" <<EOF
cn=Client CA
ca
EOF
  (set -x; certtool -s --load-privkey ca.key --template ca.tmpl --outfile ca.crt)
}

test -s "server.crt" || {
  test -e server.tmpl || cat >server.tmpl <<EOF
cn=Server
tls_www.server
EOF
  (set -x; certtool -s --load-privkey server.key --template server.tmpl --outfile server.crt)
}

test -s "client.crt" || {
  test -e client.tmpl || cat >client.tmpl <<EOF
cn=Client
tls_www_client
EOF
  (set -x; certtool -c --load-ca-certificate ca.crt --load-ca-privkey ca.key --load-privkey client.key --template client.tmpl --outfile client.crt)
}

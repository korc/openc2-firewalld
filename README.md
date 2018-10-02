# Controlling FirewallD as OpenC2 actuator

This is a test of standard under development, provided for illustration purposes only.

It is loosely based on what [`openc2-cmdgen`](https://github.com/netcoredor/openc2-cmdgen) (commit c90f08f) produces and what was on the table for WD08 of [OpenC2 Language Specification](https://github.com/oasis-tcs/openc2-oc2ls) and WD02 of [OpenC2 HTTPS implementation specification](https://github.com/oasis-tcs/openc2-impl-https).

*No warranty whatsoever.*

## Running environment

![https://mermaidjs.github.io/mermaid-live-editor/#/edit/eyJjb2RlIjoiZ3JhcGggTFJcbnN1YmdyYXBoIE9DMlByb3h5XG4gIHByeChmYTpmYS1zZXJ2ZXIgb2MyLXByb3h5LXNlcnZlcilcbiAgY3VybChmYTpmYS10ZXJtaW5hbCBjdXJsIC1kICcnJGNtZCcnIGh0dHBzOi8vLi4uKSAtLT4gfGh0dHBzIFBPU1R8cHJ4XG4gIGNtZGdlbihmYTpmYS1kZXNrdG9wIG9wZW5jMi1jbWRnZW4gYXQgaHR0cHM6Ly8uLi4vKSAtLT58aHR0cHMgUE9TVHwgcHJ4XG5lbmRcblxuc3ViZ3JhcGggRmlyZXdhbGxcbiAgY2wxKGZhOmZhLXBsdWcgZmlyZXdhbGxkLW9jMi1jbGllbnQpIC0tLXx1bml4IHNvY2tldHwgZGJ1czFcbiAgZndkMShmYTpmYS1taWNyb2NoaXAgRmlyZXdhbGxEKSAtLS0gfHVuaXggc29ja2V0fGRidXMxKGZhOmZhLWJ1bGxob3JuIEQtQnVzKVxuICBjbDEgLS0-fGh0dHBzIEdFVHwgcHJ4XG4gIGZ3ZDEgLS0-IGlwdGFibGVzKGZhOmZhLXN0cmVhbSBpcHRhYmxlcylcbiAgZndkMSAtLT4gaXBzZXQoZmE6ZmEtbGlzdCBpcHNldClcbmVuZFxuIiwibWVybWFpZCI6eyJ0aGVtZSI6ImRlZmF1bHQifX0](docs/environment-diagram.png)

### Installing

```
sudo apt install gnutls-bin golang firewalld
export GOPATH=$PWD/go
go get -u github.com/korc/openc2-firewalld/cmd/firewalld-oc2-client
go get -u github.com/korc/openc2-firewalld/cmd/oc2-proxy-server
$GOPATH/src/github.com/korc/openc2-firewalld/test/gen-certs.sh
$GOPATH/bin/oc2-proxy-server &
sudo systemctl start firewalld
sudo $GOPATH/bin/firewalld-oc2-client
```

## Command-line options

### OpenC2 command proxy server (consumer/producer)

`go run github.com/korc/openc2-firewalld/cmd/oc2-proxy-server`
- `-listen string`
    Listen address (default "localhost:1512")
- `-cert string`
    Server certificate (default "server.crt"). Empty string (`""`) will turn off TLS.
- `-key string`
    Private key for certificate (default "server.key")
- `-cacert string`
    Client CA certificate (default "ca.crt")
- `-path string`
    URL path to OpenC2 endpoint (default "/oc2")
- `-www string`
    Path to static html pages (ex: a copy of `openc2-cmdgen`)

### OpenC2 command client (consumer)

`go run github.com/korc/openc2-firewalld/cmd/firewalld-oc2-client`
- `-cert string`
    Client X509 certificate (default "client.crt")
- `-id string`
    Asset ID to use
- `-interval float`
    wait interval in seconds (default 10)
- `-key string`
    Private key for x509 certificate (default "client.key")
- `-server string`
    OpenC2 server URL (default "http://localhost:1512/oc2")
- `-zone string`
    Zone to manipulate (default "public")

### `test/gen-certs.sh`

- No options
- generates `server`, `client` and `ca` PEM-encoded `.crt` and `.key` files.
- `client.crt` will be signed by `ca.crt`.
- `xxx.tmpl` contain templates for certificates.

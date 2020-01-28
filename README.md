# Controlling FirewallD as OpenC2 actuator

This is a test of standard under development, provided for illustration purposes only.

Tries to implement [FirewallD](https://firewalld.org)-based [SLPF](https://docs.oasis-open.org/openc2/oc2slpf/v1.0/oc2slpf-v1.0.html) actuator for [OpenC2](https://docs.oasis-open.org/openc2/oc2ls/v1.0/oc2ls-v1.0.html) over [HTTPS](https://docs.oasis-open.org/openc2/open-impl-https/v1.0/open-impl-https-v1.0.html).

*No warranty whatsoever.*

## Running environment

![https://mermaidjs.github.io/mermaid-live-editor/#/edit/eyJjb2RlIjoiZ3JhcGggTFJcbnN1YmdyYXBoIE9DMlByb3h5XG4gIHByeChmYTpmYS1zZXJ2ZXIgb2MyLXByb3h5LXNlcnZlcilcbiAgY3VybChmYTpmYS10ZXJtaW5hbCBjdXJsIC1kICcnJGNtZCcnIGh0dHBzOi8vLi4uKSAtLT4gfGh0dHBzIFBPU1R8cHJ4XG4gIGNtZGdlbihmYTpmYS1kZXNrdG9wIG9wZW5jMi1jbWRnZW4gYXQgaHR0cHM6Ly8uLi4vKSAtLT58aHR0cHMgUE9TVHwgcHJ4XG5lbmRcblxuc3ViZ3JhcGggRmlyZXdhbGxcbiAgY2wxKGZhOmZhLXBsdWcgZmlyZXdhbGxkLW9jMi1jbGllbnQpIC0tLXx1bml4IHNvY2tldHwgZGJ1czFcbiAgZndkMShmYTpmYS1taWNyb2NoaXAgRmlyZXdhbGxEKSAtLS0gfHVuaXggc29ja2V0fGRidXMxKGZhOmZhLWJ1bGxob3JuIEQtQnVzKVxuICBjbDEgLS0-fGh0dHBzIEdFVHwgcHJ4XG4gIGZ3ZDEgLS0-IGlwdGFibGVzKGZhOmZhLXN0cmVhbSBpcHRhYmxlcylcbiAgZndkMSAtLT4gaXBzZXQoZmE6ZmEtbGlzdCBpcHNldClcbmVuZFxuIiwibWVybWFpZCI6eyJ0aGVtZSI6ImRlZmF1bHQifX0](docs/environment-diagram.png)

### Quick-start

```
git clone https://github.com/korc/openc2-firewalld && cd openc2-firewalld
sudo apt install gnutls-bin golang firewalld
mkdir -p run-$$ && cd run-$$
../test/gen-certs.sh
go run ../cmd/oc2-proxy-server -cmdschema ../test/command-schema.json -respschema ../test/response-schema.json &
sudo systemctl start firewalld && sudo go run ../cmd/firewalld-oc2-client &
../test/test-request.sh ../test/test-query.json
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

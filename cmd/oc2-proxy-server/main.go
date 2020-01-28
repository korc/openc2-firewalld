package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"github.com/korc/openc2-firewalld"
	"github.com/santhosh-tekuri/jsonschema"
	jsonSchemaDecoders "github.com/santhosh-tekuri/jsonschema/decoders"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	listenAddr := flag.String("listen", "localhost:1512", "Listen address")
	oc2path := flag.String("path", "/oc2", "URL path to OpenC2 endpoint")
	staticWWW := flag.String("www", "", "Path to static html pages")
	certFile := flag.String("cert", "server.crt", "Server certificate")
	keyFile := flag.String("key", "server.key", "Private key for certificate")
	caCertFile := flag.String("cacert", "ca.crt", "Client CA certificate")
	cmdSchemaFile := flag.String("cmdschema", "", "Commands JSON-schema file")
	respSchemaFile := flag.String("respschema", "", "Responses JSON-schema file")
	flag.Parse()
	mplx := NewOpenC2RequestMultiplexer()
	jsonSchemaDecoders.Register("base16", hex.DecodeString)
	if *cmdSchemaFile != "" {
		if sch, err := jsonschema.Compile(*cmdSchemaFile); err != nil {
			log.Fatalf("Cannot read commands JSON schema from %#v: %s", *cmdSchemaFile, err)
		} else {
			mplx.cmdSchema = sch
		}
	}
	if *respSchemaFile != "" {
		if sch, err := jsonschema.Compile(*cmdSchemaFile); err != nil {
			log.Fatalf("Cannot read response JSON schema from %#v: %s", *respSchemaFile, err)
		} else {
			mplx.respSchema = sch
		}
	}
	http.Handle(*oc2path, mplx)
	http.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(struct {
			Commands []*openc2.OpenC2Command
			Assets   map[string]*openC2AssetRecord
		}{mplx.commandQueue, mplx.assets}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Print("Error: ", err)
			w.Write([]byte("Bad things happen."))
		}
	})
	if *staticWWW != "" {
		http.Handle("/", http.FileServer(http.Dir(*staticWWW)))
	}
	log.Printf("Listening and serving on: %s", *listenAddr)
	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("Cannot listen on %#v: %s", *listenAddr, err)
	}
	if *certFile != "" {
		if *keyFile == "" {
			*keyFile = *certFile
		}
		crt, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Cannot load cert/key from %#v and %#v: %s", *certFile, *keyFile, err)
		}
		tlsConfig := &tls.Config{Certificates: []tls.Certificate{crt}}
		if *caCertFile != "" {
			pemData, err := ioutil.ReadFile(*caCertFile)
			if err != nil {
				log.Fatalf("Cannot read CA certs from %#v: %s", *caCertFile, err)
			}
			tlsConfig.ClientCAs = x509.NewCertPool()
			if !tlsConfig.ClientCAs.AppendCertsFromPEM(pemData) {
				log.Fatal("Could not add root CA certificates")
			}
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			tlsConfig.ClientAuth = tls.RequestClientCert
		}
		listener = tls.NewListener(listener, tlsConfig)
		log.Printf("SSL enabled, cert=%s", *certFile)
	}
	log.Fatal(http.Serve(listener, &LoggingHandler{Handler: http.DefaultServeMux}))
}

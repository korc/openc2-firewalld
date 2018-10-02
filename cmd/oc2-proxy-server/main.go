package main

import (
	"encoding/json"
	"flag"
	"github.com/korc/openc2-firewalld"
	"log"
	"net/http"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	listenAddr := flag.String("listen", "localhost:1512", "Listen address")
	oc2path := flag.String("path", "/oc2", "URL path to OpenC2 endpoint")
	staticWWW := flag.String("www", "", "Path to static html pages")
	flag.Parse()
	mplx := NewOpenC2RequestMultiplexer()
	http.Handle(*oc2path, mplx)
	http.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(struct {
			Commands     []*openc2.OpenC2Command
			Assets       map[string]*openC2AssetRecord
			RequestCount int64
		}{mplx.commandQueue, mplx.assets, mplx.reqCount}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Print("Error: ", err)
			w.Write([]byte("Bad things happen."))
		}
	})
	if *staticWWW != "" {
		http.Handle("/", http.FileServer(http.Dir(*staticWWW)))
	}
	log.Printf("Listening and serving on: %s", *listenAddr)
	log.Fatal(http.ListenAndServe(*listenAddr, &LoggingHandler{Handler: http.DefaultServeMux}))
}

package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/korc/openc2-firewalld"
)

type openC2AssetRecord struct {
	lastAccess time.Time
	queueIndex int
}

type OpenC2RequestMultiplexer struct {
	commandQueue []*openc2.OpenC2Command
	assets       map[string]*openC2AssetRecord
	reqCount     int64
	modReq       *sync.Mutex
}

func (rqm *OpenC2RequestMultiplexer) handleCORSOptions(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if acrm := r.Header.Get("Access-Control-Request-Method"); r.Method == "OPTIONS" && origin != "" && acrm != "" {
		log.Printf("CORS access control allowed from %s", origin)
		w.Header().Add("Access-Control-Allow-Origin", origin)
		w.Header().Add("Access-Control-Allow-Methods", "*")
		w.Header().Add("Access-Control-Allow-Headers", "*")
		w.Header().Add("Vary", "Origin")
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
	return
}

func (rqm *OpenC2RequestMultiplexer) handlePost(w http.ResponseWriter, r *http.Request) {
	body := make([]byte, 0)
	for {
		data := make([]byte, 8192)
		nRead, err := r.Body.Read(data)
		// log.Printf("Body read: nRead=%#v data=%#v err=%#v", nRead, data, err)
		if nRead > 0 {
			body = append(body, data[:nRead]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Print("Read error: ", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Read error"))
			return
		}
	}
	w.Header().Add("Content-Type", openc2.OpenC2ResponseType)
	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Add("Access-Control-Allow-Origin", origin)
	}
	var oc2cmd *openc2.OpenC2Command
	log.Printf("Unmarshalling: %#v", string(body))
	if err := json.Unmarshal(body, &oc2cmd); err != nil {
		log.Print("Unmarshal error: ", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid data"))
		return
	}
	rqm.commandQueue = append(rqm.commandQueue, oc2cmd)
	w.WriteHeader(http.StatusOK)
	d, _ := json.Marshal(openc2.OpenC2Response{Status: 200, StatusText: "Command added to the queue."})
	w.Write([]byte(d))
	log.Printf("Added to queue: %#v, %d assets listening: %#v", oc2cmd, len(rqm.assets), rqm.assets)
	return
}

func (rqm *OpenC2RequestMultiplexer) handleGet(w http.ResponseWriter, r *http.Request) {
	var asset *openC2AssetRecord
	var assetID string
	useTLS := r.TLS != nil && len(r.TLS.PeerCertificates) > 0
	if useTLS {
		assetID = base64.RawURLEncoding.EncodeToString(r.TLS.PeerCertificates[0].RawSubject)
	} else {
		assetID = r.Header.Get(openc2.OpenC2AssetIDHeader)
	}
	if gotAsset, ok := rqm.assets[assetID]; !ok {
		if !useTLS || assetID == "" {
			assetID = RandStringBytes(16)
		}
		asset = &openC2AssetRecord{lastAccess: time.Now(), queueIndex: 0}
		rqm.assets[assetID] = asset
		w.Header().Set(openc2.OpenC2AssetIDHeader, assetID)
		log.Printf("Created new asset ID: %#v", assetID)
	} else {
		asset = gotAsset
		log.Printf("Asset ID: %#v", assetID)
	}

	asset.lastAccess = time.Now()
	if len(rqm.commandQueue) > asset.queueIndex {
		nextCommand := rqm.commandQueue[asset.queueIndex]
		commandData, err := json.Marshal(nextCommand)
		if err != nil {
			log.Printf("Cannot marshal command to data %#v: %s", nextCommand, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		asset.queueIndex = asset.queueIndex + 1
		w.Header().Set("Content-Type", openc2.OpenC2CommandType)
		w.WriteHeader(http.StatusOK)
		w.Write(commandData)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (rqm *OpenC2RequestMultiplexer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "OPTIONS":
		rqm.handleCORSOptions(w, r)
		return
	case "POST":
		rqm.handlePost(w, r)
		return
	case "GET":
		rqm.handleGet(w, r)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("Don't know how to process that."))
}

func NewOpenC2RequestMultiplexer() (rqm *OpenC2RequestMultiplexer) {
	rqm = &OpenC2RequestMultiplexer{}
	rqm.commandQueue = make([]*openc2.OpenC2Command, 0)
	rqm.modReq = &sync.Mutex{}
	rqm.assets = make(map[string]*openC2AssetRecord)
	return
}

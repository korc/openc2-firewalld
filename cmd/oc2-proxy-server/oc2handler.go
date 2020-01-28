package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/korc/openc2-firewalld"
	"github.com/santhosh-tekuri/jsonschema"
)

type openC2AssetRecord struct {
	LastAccess time.Time
	QueueIndex int
}

type OpenC2RequestMultiplexer struct {
	commandQueue []*openc2.OpenC2Command
	assets       map[string]*openC2AssetRecord
	modReq       *sync.Mutex
	cmdSchema    *jsonschema.Schema
	respSchema   *jsonschema.Schema
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
	if requestId := r.Header.Get(openc2.OpenC2RequestIDHeader); requestId != "" {
		w.Header().Set(openc2.OpenC2RequestIDHeader, requestId)
	}
	if ctype := r.Header.Get("Content-Type"); ctype != openc2.OpenC2CommandType {
		log.Printf("Wrong Content-Type header: %#v", ctype)
		rqm.sendOpenC2Response(w, openc2.OpenC2Response{Status: openc2.StatusBadRequest,
			StatusText: fmt.Sprintf("Wrong Content-Type: %#v, expected %#v", ctype, openc2.OpenC2CommandType)})
		return
	}
	if rqm.cmdSchema != nil {
		if err := rqm.cmdSchema.Validate(bytes.NewReader(body)); err != nil {
			log.Printf("Schema validation failed: %s", err)
			rqm.sendOpenC2Response(w, openc2.OpenC2Response{Status: openc2.StatusBadRequest,
				StatusText: fmt.Sprintf("Data not compliant to schema:\n%s", err)})
			return
		}
	}
	var oc2cmd *openc2.OpenC2Command
	log.Printf("Unmarshalling: %#v", string(body))
	if err := json.Unmarshal(body, &oc2cmd); err != nil {
		log.Print("Unmarshal error: ", err)
		rqm.sendOpenC2Response(w, openc2.OpenC2Response{Status: openc2.StatusNotImplemented, StatusText: "Can't unmarshal that"})
		return
	}
	rqm.modReq.Lock()
	rqm.commandQueue = append(rqm.commandQueue, oc2cmd)
	rqm.modReq.Unlock()
	if oc2cmd.Action == openc2.ActionQuery {
		rqm.handleActionQuery(w, oc2cmd)
		return
	}
	sendResponse := true
	if args, ok := oc2cmd.Args.(map[string]interface{}); ok {
		if rr, ok := args["response_requested"]; ok {
			log.Printf("Response Requested: %#v", rr)
			if rr == "none" {
				sendResponse = false
			}
		} else {
			log.Printf("No response requested")
		}
	}
	if sendResponse {
		rqm.sendOpenC2Response(w, openc2.OpenC2Response{Status: openc2.StatusOK, StatusText: "Command added to the queue."})
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
	log.Printf("Added to queue: %#v, %d assets listening: %#v", oc2cmd, len(rqm.assets), rqm.assets)
	return
}

var supportedTargets = []openc2.OpenC2TargetType{
	openc2.TargetTypeIPv4Net,
	openc2.TargetTypeIPv4Connection,
	openc2.TargetTypeIPv6Net,
	openc2.TargetTypeIPv6Connection,
}

func (rqm *OpenC2RequestMultiplexer) handleActionQuery(w http.ResponseWriter, cmd *openc2.OpenC2Command) {
	if args, haveArgs := cmd.Args.(map[string]interface{}); haveArgs {
		if rr, haveRR := args["response_requested"]; haveRR {
			if rr != "complete" {
				log.Printf("failed action=query arguments.response_requested check")
				rqm.sendOpenC2Response(w, openc2.OpenC2Response{Status: openc2.StatusBadRequest, StatusText: "response_requested != 'complete'"})
				return
			}
		}
	}
	resp := openc2.OpenC2Response{Status: openc2.StatusOK}
	if target, haveTarget := cmd.Target.(openc2.OpenC2GenericTarget); haveTarget {
		if features, haveFeatures := target["features"]; haveFeatures {
			if flist, haveFList := features.([]interface{}); haveFList {
				for _, f := range flist {
					switch f {
					case "versions":
						resp.AddResults("versions", []string{"1.0"})
					case "profiles":
						resp.AddResults("profiles", []string{"slpf"})
					case "pairs":
						resp.AddResults("pairs", map[string]interface{}{
							"allow": supportedTargets,
							"deny":  supportedTargets,
							"query": []string{"features"},
						})
					default:
						log.Printf("WARNING: Unkonwn features in query: %#v", f)

					}
				}
			}
		}
	}
	log.Printf("resp: %#v", resp)
	rqm.sendOpenC2Response(w, resp)
}

func (rqm *OpenC2RequestMultiplexer) sendOpenC2Response(w http.ResponseWriter, resp openc2.OpenC2Response) {
	st := resp.Status
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Cannot marshal response: %s", err)
		st = http.StatusInternalServerError
		data = []byte("Error")
	}
	w.Header().Set("Cache-control", "no-cache")
	w.WriteHeader(int(st))
	w.Write(data)
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
	rqm.modReq.Lock()
	if gotAsset, ok := rqm.assets[assetID]; !ok {
		if !useTLS || assetID == "" {
			assetID = RandStringBytes(16)
		}
		asset = &openC2AssetRecord{LastAccess: time.Now(), QueueIndex: len(rqm.commandQueue)}
		rqm.assets[assetID] = asset
		rqm.modReq.Unlock()
		w.Header().Set(openc2.OpenC2AssetIDHeader, assetID)
		log.Printf("Created new asset ID: %#v", assetID)
	} else {
		asset = gotAsset
		rqm.modReq.Unlock()
		log.Printf("Asset ID: %#v", assetID)
	}

	asset.LastAccess = time.Now()
	if len(rqm.commandQueue) > asset.QueueIndex {
		nextCommand := rqm.commandQueue[asset.QueueIndex]
		commandData, err := json.Marshal(nextCommand)
		if err != nil {
			log.Printf("Cannot marshal command to data %#v: %s", nextCommand, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		asset.QueueIndex = asset.QueueIndex + 1
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

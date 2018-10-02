package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/korc/openc2-firewalld"
)

const clientVersion = "0.1"
const userAgent = "OpenC2-FirewallD-Client/" + clientVersion

func main() {
	fwdctrl, err := NewFirewallDControl()
	if err != nil {
		log.Fatal("Could not get FirewallD control: ", err)
	}

	server := flag.String("server", "http://localhost:1512/oc2", "OpenC2 server URL")
	zone := flag.String("zone", fwdctrl.Zone, "Zone to manipulate")
	assetID := flag.String("id", "", "Asset ID to use")
	waitIntervalFlag := flag.Float64("interval", 10, "wait interval in seconds")

	flag.Parse()
	if *zone != fwdctrl.Zone {
		log.Printf("FW zone set to %s", *zone)
		fwdctrl.Zone = *zone
	}

	for {
		waitIntervalDelay := time.Nanosecond * time.Duration(int(*waitIntervalFlag*10e8))
		req, err := http.NewRequest("GET", *server, nil)
		if err != nil {
			log.Fatal("Cannot create request: ", err)
		}
		if *assetID != "" {
			req.Header.Set(openc2.OpenC2AssetIDHeader, *assetID)
		}
		req.Header.Set("Accept", openc2.OpenC2CommandType)
		req.Header.Set("User-Agent", userAgent)
		log.Printf("Sending request for data (asset-id=%#v)", *assetID)
		if resp, err := http.DefaultClient.Do(req); err != nil {
			log.Printf("Error getting data from OpenC2 server: %s", err)
		} else {
			log.Printf("Response from server: %#v", resp)
			if responseAssetID := resp.Header.Get("X-OpenC2-Asset-Id"); responseAssetID != "" {
				log.Printf("Asset ID set to %#v", responseAssetID)
				*assetID = responseAssetID
			}
			if resp.ContentLength > 0 {
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Print("Cannot read body: ", err)
				} else {
					var oc2cmd openc2.OpenC2Command
					if err := json.Unmarshal(body, &oc2cmd); err != nil {
						log.Printf("Failed to parse response: %s: %#v", err, string(body))
					} else {
						fwdctrl.OpenC2Act(oc2cmd)
						waitIntervalDelay = 0
					}
				}
			}
		}
		if waitIntervalDelay > 0 {
			log.Printf("Sleeping %s before next run..", waitIntervalDelay)
			time.Sleep(waitIntervalDelay)

		}
	}
}
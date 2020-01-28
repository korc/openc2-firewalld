package openc2

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"log"
	"path"
	"strings"
	"testing"

	"encoding/json"

	"github.com/santhosh-tekuri/jsonschema"
	jsonSchemaDecoders "github.com/santhosh-tekuri/jsonschema/decoders"
)

func readFiles(directoryName, prefix, suffix string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	if results, err := ioutil.ReadDir(directoryName); err != nil {
		return nil, err
	} else {
		for _, fn := range results {
			if fn.IsDir() {
				continue
			}
			fname := fn.Name()
			if !strings.HasPrefix(fname, prefix) || !strings.HasSuffix(fname, suffix) {
				continue
			}
			if cmd, err := ioutil.ReadFile(path.Join(directoryName, fname)); err != nil {
				log.Printf("Error reading %#v: %s", fn, err)
			} else {
				files[fname[len(prefix):len(fname)-len(suffix)]] = cmd
			}
		}
	}
	return files, nil
}

var reqTestDir = "test"

func TestRequests(t *testing.T) {
	var oc2cmd *OpenC2Command
	commands, err := readFiles(reqTestDir, "test-", ".json")
	if err != nil {
		log.Fatalf("Cannot read commands from %#v: %s", reqTestDir, err)
	}
	jsonSchemaDecoders.Register("base16", hex.DecodeString)
	cs, err := jsonschema.Compile(path.Join(reqTestDir, "command-schema.json"))
	if err != nil {
		log.Fatalf("Cannot load schema: %s", err)
	}
	for name, cmd := range commands {
		log.Printf("Original JSON of %#v:\n%s", name, cmd)
		if err := json.Unmarshal(cmd, &oc2cmd); err != nil {
			t.Errorf("Could not parse %#v: %s", name, err)
			continue
		}
		if err := cs.Validate(bytes.NewReader(cmd)); err != nil {
			t.Errorf("Could not validate schema for %#v: %s", name, err)
		} else {
			log.Printf("Schema OK")
		}
		if jsonBytes, err := json.MarshalIndent(oc2cmd, "", "  "); err != nil {
			t.Errorf("Could not marshal %#v back to JSON: %s", name, err)
			continue
		} else {
			log.Printf("JSON of %#v:\n%s", name, string(jsonBytes))
		}
	}
}

func TestResults(t *testing.T) {
	var oc2resp *OpenC2Response
	responses, err := readFiles(reqTestDir, "resp-", ".json")
	if err != nil {
		log.Fatalf("Cannot read commands from %#v: %s", reqTestDir, err)
	}
	jsonSchemaDecoders.Register("base16", hex.DecodeString)
	cs, err := jsonschema.Compile(path.Join(reqTestDir, "response-schema.json"))
	if err != nil {
		log.Fatalf("Cannot load response schema: %s", err)
	}
	for name, resp := range responses {
		if err := json.Unmarshal([]byte(resp), &oc2resp); err != nil {
			t.Errorf("Could not parse %#v: %s", name, err)
			continue
		}
		log.Printf("parsed %#v -> %#v", name, oc2resp)
		if err := cs.Validate(bytes.NewReader(resp)); err != nil {
			t.Errorf("Could not validate response schema for %#v: %s", name, err)
		} else {
			log.Printf("Schema OK")
		}
		if jsonBytes, err := json.MarshalIndent(oc2resp, "", "  "); err != nil {
			t.Errorf("Could not marshal %#v to JSON: %s", name, err)
			continue
		} else {
			log.Printf("JSON of %#v:\n%s", name, string(jsonBytes))
		}
	}
}

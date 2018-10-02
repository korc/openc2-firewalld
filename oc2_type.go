package openc2

import (
	"encoding/json"
	"fmt"
	"log"
)

type OpenC2Response struct {
	IdRef      string `json:"id_ref"`
	Status     int
	StatusText string `json:"status_text,omitempty"`
	Results    interface{}
}

type OpenC2Command struct {
	Action   string      `json:"action"`
	Target   interface{} `json:"target"`
	Id       string      `json:"id,omitempty"`
	Args     interface{} `json:"args,omitempty"`
	Actuator interface{} `json:"actuator,omitempty"`
}

type OpenC2IPConnection struct {
	Layer4Protocol     string `json:"layer4_protocol,omitempty"`
	SourceAddress      string `json:"src_addr,omitempty"`
	SourcePort         string `json:"src_port,omitempty"`
	DestinationAddress string `json:"dst_addr,omitempty"`
	DestinationPort    string `json:"dst_port,omitempty"`
}

func (c *OpenC2Command) MarshalJSON() ([]byte, error) {
	type F OpenC2Command
	out := &struct {
		Target interface{} `json:"target"`
		*F
	}{F: (*F)(c)}
	switch c.Target.(type) {
	case *OpenC2IPConnection:
		out.Target = map[string]interface{}{"ip-connection": c.Target}
	default:
		out.Target = c.Target
		log.Printf("Serializing unknown target: %#v", c.Target)
	}
	return json.Marshal(out)
}

func (c *OpenC2Command) UnmarshalJSON(b []byte) error {
	type X OpenC2Command
	x := &struct {
		Target json.RawMessage `json:"target"`
		*X
	}{X: (*X)(c)}
	if err := json.Unmarshal(b, &x); err != nil {
		return err
	}
	tgt := make(map[string]json.RawMessage)
	if err := json.Unmarshal(x.Target, &tgt); err != nil {
		log.Printf("Could not unmarshal target to map[string]json.RawMessage")
		return fmt.Errorf("", err)
	}
	if ipc, ok := tgt["ip-connection"]; ok {
		log.Print("Target is IP connection")
		c.Target = &OpenC2IPConnection{}
		if err := json.Unmarshal(ipc, &c.Target); err != nil {
			return err
		}
	} else {
		unknownTarget := make(map[string]interface{})
		for key, rawValue := range tgt {
			var parsedValue interface{}
			if err := json.Unmarshal(rawValue, &parsedValue); err != nil {
				return err
			}
			unknownTarget[key] = parsedValue
		}
		c.Target = unknownTarget
	}
	log.Printf("Unmarshalled: %#v", c)
	return nil
}

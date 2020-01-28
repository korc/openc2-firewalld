package openc2

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
)

type OpenC2Status int

const (
	StatusProcessing         OpenC2Status = 102
	StatusOK                 OpenC2Status = 200
	StatusBadRequest         OpenC2Status = 400
	StatusUnauthorized       OpenC2Status = 401
	StatusForbidden          OpenC2Status = 403
	StatusNotFound           OpenC2Status = 404
	StatusInternalError      OpenC2Status = 500
	StatusNotImplemented     OpenC2Status = 501
	StatusServiceUnavailable OpenC2Status = 503
)

type OpenC2Response struct {
	Status     OpenC2Status           `json:"status"`
	StatusText string                 `json:"status_text,omitempty"`
	Results    map[string]interface{} `json:"results,omitempty"`
}

func (r *OpenC2Response) AddResults(name string, data interface{}) {
	if r.Results == nil {
		r.Results = make(map[string]interface{})
	}
	r.Results[name] = data
}

type OpenC2Action string

const (
	ActionScan        OpenC2Action = "scan"
	ActionLocate      OpenC2Action = "locate"
	ActionQuery       OpenC2Action = "query"
	ActionDeny        OpenC2Action = "deny"
	ActionContain     OpenC2Action = "contain"
	ActionAllow       OpenC2Action = "allow"
	ActionStart       OpenC2Action = "start"
	ActionStop        OpenC2Action = "stop"
	ActionRestart     OpenC2Action = "restart"
	ActionCancel      OpenC2Action = "cancel"
	ActionSet         OpenC2Action = "set"
	ActionUpdate      OpenC2Action = "update"
	ActionRedirect    OpenC2Action = "redirect"
	ActionCreate      OpenC2Action = "create"
	ActionDelete      OpenC2Action = "delete"
	ActionDetonate    OpenC2Action = "detonate"
	ActionRestore     OpenC2Action = "restore"
	ActionCopy        OpenC2Action = "copy"
	ActionInvestigate OpenC2Action = "investigate"
	ActionRemediate   OpenC2Action = "remediate"
)

type OpenC2TargetType string

const (
	TargetTypeArtifact       OpenC2TargetType = "artifact"
	TargetTypeCommand        OpenC2TargetType = "command"
	TargetTypeDevice         OpenC2TargetType = "device"
	TargetTypeDomainName     OpenC2TargetType = "domain_name"
	TargetTypeEmailAddr      OpenC2TargetType = "email_addr"
	TargetTypeFeatures       OpenC2TargetType = "features"
	TargetTypeFile           OpenC2TargetType = "file"
	TargetTypeIdnDomainName  OpenC2TargetType = "idn_domain_name"
	TargetTypeIdnEmailAddr   OpenC2TargetType = "idn_email_addr"
	TargetTypeIPv4Net        OpenC2TargetType = "ipv4_net"
	TargetTypeIPv6Net        OpenC2TargetType = "ipv6_net"
	TargetTypeIPv4Connection OpenC2TargetType = "ipv4_connection"
	TargetTypeIPv6Connection OpenC2TargetType = "ipv6_connection"
	TargetTypeIri            OpenC2TargetType = "iri"
	TargetTypeMacAddr        OpenC2TargetType = "mac_addr"
	TargetTypeProcess        OpenC2TargetType = "process"
	TargetTypeProperties     OpenC2TargetType = "properties"
	TargetTypeURI            OpenC2TargetType = "uri"
)

type OpenC2Command struct {
	Action   OpenC2Action `json:"action"`
	Target   interface{}  `json:"target"`
	ID       string       `json:"id,omitempty"`
	Args     interface{}  `json:"args,omitempty"`
	Actuator interface{}  `json:"actuator,omitempty"`
}

type TargetIPConnection struct {
	Protocol           string `json:"protocol,omitempty"`
	SourceAddress      string `json:"src_addr,omitempty"`
	SourcePort         int    `json:"src_port,omitempty"`
	DestinationAddress string `json:"dst_addr,omitempty"`
	DestinationPort    int    `json:"dst_port,omitempty"`
}

type TargetIPv4Connection struct{ TargetIPConnection }
type TargetIPv6Connection struct{ TargetIPConnection }

type TargetIPv4Net string
type TargetIPv6Net string

type OpenC2GenericTarget map[OpenC2TargetType]interface{}

func (c *OpenC2Command) MarshalJSON() ([]byte, error) {
	type F OpenC2Command
	out := &struct {
		Target interface{} `json:"target"`
		*F
	}{F: (*F)(c)}
	switch tgt := c.Target.(type) {
	case *TargetIPv4Connection:
		out.Target = map[OpenC2TargetType]interface{}{TargetTypeIPv4Connection: tgt}
	case *TargetIPv6Connection:
		out.Target = map[OpenC2TargetType]interface{}{TargetTypeIPv6Connection: tgt}
	case *TargetIPv4Net:
		out.Target = map[OpenC2TargetType]interface{}{TargetTypeIPv4Net: tgt}
	case *TargetIPv6Net:
		out.Target = map[OpenC2TargetType]interface{}{TargetTypeIPv6Net: tgt}
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
	tgt := make(map[OpenC2TargetType]json.RawMessage)
	if err := json.Unmarshal(x.Target, &tgt); err != nil {
		log.Printf("Could not unmarshal target to map[string]json.RawMessage")
		return fmt.Errorf("%s", err)
	}
	if ipv4conn, ok := tgt[TargetTypeIPv4Connection]; ok {
		log.Print("Target is IPv4 connection")
		c.Target = &TargetIPv4Connection{}
		if err := json.Unmarshal(ipv4conn, &c.Target); err != nil {
			return err
		}
	} else if ipv6conn, ok := tgt[TargetTypeIPv6Connection]; ok {
		log.Print("Target is IPv6 connection")
		c.Target = &TargetIPv6Connection{}
		if err := json.Unmarshal(ipv6conn, &c.Target); err != nil {
			return err
		}
	} else if ipv4net, ok := tgt[TargetTypeIPv4Net]; ok {
		log.Print("Target is IPv4 net")
		c.Target = new(TargetIPv4Net)
		if err := json.Unmarshal(ipv4net, &c.Target); err != nil {
			return err
		}
	} else if ipv6net, ok := tgt[TargetTypeIPv6Net]; ok {
		log.Print("Target is IPv6 net")
		c.Target = new(TargetIPv6Net)
		if err := json.Unmarshal(ipv6net, &c.Target); err != nil {
			return err
		}
	} else {
		log.Printf("Target type unknown: keys=%#v", reflect.ValueOf(tgt).MapKeys())
		unknownTarget := make(OpenC2GenericTarget)
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

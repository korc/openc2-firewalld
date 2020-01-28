package main

import (
	"fmt"
	"github.com/godbus/dbus"
	"github.com/korc/openc2-firewalld"
	"log"
	"strings"
)

const fwd1Interface = "org.fedoraproject.FirewallD1"
const fwd1Path = "/org/fedoraproject/FirewallD1"

type FwDPolicy string

const (
	FwDPolicyAccept         FwDPolicy = "accept"
	FwDPolicyDrop           FwDPolicy = "drop"
	FwDPolicyReject         FwDPolicy = "reject"
	FwDPolicyRejectFalseAck FwDPolicy = "reject type=\"tcp-reset\""
)

type FirewallDControl struct {
	Connection *dbus.Conn
	FwD1       dbus.BusObject
	Zone       string
}

type FirewallDRule struct {
	policy             FwDPolicy
	family             string
	sourcePort         int
	destinationPort    int
	sourceAddress      string
	destinationAddress string
	protocol           string
	dropProcess        string
}

func NewFirewallDRule(policy FwDPolicy) *FirewallDRule {
	return &FirewallDRule{policy: policy}
}

func (fwr *FirewallDRule) String() string {
	ruleParts := []string{"rule"}
	if fwr.sourceAddress != "" {
		ruleParts = append(ruleParts, fmt.Sprintf("source address=\"%s\"", fwr.sourceAddress))
	}
	if fwr.sourcePort != 0 {
		ruleParts = append(ruleParts, fmt.Sprintf("source-port port=\"%d\"", fwr.sourcePort))
		ruleParts = append(ruleParts, fmt.Sprintf("protocol=\"%s\"", fwr.protocol))
	}
	if fwr.destinationAddress != "" {
		ruleParts = append(ruleParts, fmt.Sprintf("destination address=\"%s\"", fwr.destinationAddress))
	}
	if fwr.destinationPort != 0 {
		ruleParts = append(ruleParts, fmt.Sprintf("port port=\"%d\"", fwr.destinationPort))
		ruleParts = append(ruleParts, fmt.Sprintf("protocol=\"%s\"", fwr.protocol))
	}
	if fwr.family != "" {
		ruleParts = append(ruleParts, fmt.Sprintf("family=\"%s\"", fwr.family))
	}
	ruleParts = append(ruleParts, string(fwr.policy))
	return strings.Join(ruleParts, " ")
}

func (fwr *FirewallDRule) ProcessOC2Target(target interface{}) error {
	switch tgt := target.(type) {
	case *openc2.TargetIPv6Connection:
		fwr.family = "ipv6"
		target = &tgt.TargetIPConnection
	case *openc2.TargetIPv4Connection:
		fwr.family = "ipv4"
		target = &tgt.TargetIPConnection
	}
	switch tgt := target.(type) {
	case *openc2.TargetIPConnection:
		fwr.sourceAddress = tgt.SourceAddress
		fwr.destinationAddress = tgt.DestinationAddress
		fwr.protocol = tgt.Protocol
		fwr.sourcePort = tgt.SourcePort
		fwr.destinationPort = tgt.DestinationPort
	case *openc2.TargetIPv6Net:
		fwr.family = "ipv6"
		fwr.sourceAddress = string(*tgt)
	case *openc2.TargetIPv4Net:
		fwr.family = "ipv4"
		fwr.sourceAddress = string(*tgt)
	default:
		return UnknownTargetError
	}
	return nil
}

func NewFirewallDControl() (*FirewallDControl, error) {
	ret := &FirewallDControl{}
	bus, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}
	ret.Connection = bus

	ret.FwD1 = ret.Connection.Object(fwd1Interface, fwd1Path)

	if err = ret.FwD1.Call(fwd1Interface+".getDefaultZone", 0).Store(&ret.Zone); err != nil {
		return nil, err
	}

	return ret, nil
}

func (fwd *FirewallDControl) AddIC2Rule(rule *FirewallDRule) (string, error) {
	var callRet string
	log.Printf("Adding rule: %s", rule)
	if err := fwd.FwD1.Call(fwd1Interface+".zone.addRichRule", 0, fwd.Zone, rule.String(), 0).Store(&callRet); err != nil {
		log.Printf("Adding rich rule %#v failed: %s", rule, err)
		return callRet, err
	}
	return callRet, nil
}

func (fwd *FirewallDControl) OpenC2Act(oc2cmd openc2.OpenC2Command) error {
	log.Printf("Command: %#v", oc2cmd)
	var policy FwDPolicy
	switch oc2cmd.Action {
	case openc2.ActionDeny:
		policy = FwDPolicyReject
		if args, haveArgs := oc2cmd.Args.(map[string]interface{}); haveArgs {
			log.Printf("slpf arg")
			if slpf, haveSlpf := args["slpf"]; haveSlpf {
				if slpfMap, haveSlpfMap := slpf.(map[string]interface{}); haveSlpfMap {
					if dropProcess, haveDP := slpfMap["drop_process"]; haveDP {
						switch dropProcess {
						case "none":
							policy = FwDPolicyDrop
						case "false_ack":
							policy = FwDPolicyRejectFalseAck
						case "reject":
							policy = FwDPolicyReject
						default:
							log.Printf("WARNING: unknown drop_process: %#v", dropProcess)
						}
					}
				}
			}
		}
	case openc2.ActionAllow:
		policy = FwDPolicyAccept
	default:
		log.Printf("Don't know what to do with action %#v", oc2cmd.Action)
		return UnknownActionError
	}
	rule := NewFirewallDRule(policy)
	if err := rule.ProcessOC2Target(oc2cmd.Target); err != nil {
		log.Printf("Cannot process target %#v: %s", oc2cmd.Target, err)
		return err
	}
	if res, err := fwd.AddIC2Rule(rule); err != nil {
		return err
	} else {
		log.Printf("Action done: %#v", res)
	}
	return nil
}

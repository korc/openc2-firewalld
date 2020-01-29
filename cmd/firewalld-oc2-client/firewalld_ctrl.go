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
	ruleIdMap  map[float64]*FirewallDRule
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
	ret.ruleIdMap = make(map[float64]*FirewallDRule)

	return ret, nil
}

func (fwd *FirewallDControl) RemoveIC2Rule(rule *FirewallDRule) (callRet string, err error) {
	log.Printf("Removing rule: %s", rule)
	if err = fwd.FwD1.Call(fwd1Interface+".zone.removeRichRule", 0, fwd.Zone, rule.String()).Store(&callRet); err != nil {
		log.Printf("Could not remove rule %#v: %s", rule, err)
		return callRet, err
	}
	return callRet, nil
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
	var ruleId float64
	haveRuleId := false
	argsSlpf := make(map[string]interface{})
	if args, ok := oc2cmd.Args.(map[string]interface{}); ok {
		if slpf, ok := args["slpf"]; ok {
			if slpfMap, ok := slpf.(map[string]interface{}); ok {
				argsSlpf = slpfMap
			}
		}
	}
	switch oc2cmd.Action {
	case openc2.ActionDeny:
		policy = FwDPolicyReject
		if dropProcess, ok := argsSlpf["drop_process"]; ok {
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
		if insertRule, ok := argsSlpf["insert_rule"]; ok {
			if insertRuleFloat, ok := insertRule.(float64); ok {
				ruleId = insertRuleFloat
				haveRuleId = true
			}
		}
	case openc2.ActionAllow:
		policy = FwDPolicyAccept
	case openc2.ActionDelete:
		if target, ok := oc2cmd.Target.(openc2.OpenC2GenericTarget); ok {
			if slpfRuleNumber, ok := target["slpf:rule_number"]; ok {
				if slpfRuleNumberInt, ok := slpfRuleNumber.(float64); ok {
					if rule, ok := fwd.ruleIdMap[slpfRuleNumberInt]; ok {
						if callret, err := fwd.RemoveIC2Rule(rule); err != nil {
							return err
						} else {
							log.Printf("Removing rule OK: %s", callret)
							delete(fwd.ruleIdMap, slpfRuleNumberInt)
							return nil
						}
					} else {
						return InvalidRuleNumber
					}
				} else {
					return UnknownTargetError
				}
			} else {
				return UnknownTargetError
			}
		} else {
			return UnknownTargetError
		}
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
		if haveRuleId {
			fwd.ruleIdMap[ruleId] = rule
		}
	}
	return nil
}

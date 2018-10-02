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

type FirewallDControl struct {
	Connection *dbus.Conn
	FwD1       dbus.BusObject
	Zone       string
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

func (fwd *FirewallDControl) AddIC2Rule(policy string, ic *openc2.OpenC2IPConnection) (string, error) {
	var rule, callRet string
	ruleParts := []string{"rule"}
	if ic.SourceAddress != "" {
		ruleParts = append(ruleParts, fmt.Sprintf("source address=\"%s\"", ic.SourceAddress))
	}
	if ic.SourcePort != "" {
		ruleParts = append(ruleParts, fmt.Sprintf("source-port port=\"%s\"", ic.SourcePort))
		ruleParts = append(ruleParts, fmt.Sprintf("protocol=\"%s\"", ic.Layer4Protocol))
	}
	if ic.DestinationAddress != "" {
		ruleParts = append(ruleParts, fmt.Sprintf("destination address=\"%s\"", ic.DestinationAddress))
	}
	if ic.DestinationPort != "" {
		if strings.Trim(ic.DestinationPort, "0123456789") == "" {
			ruleParts = append(ruleParts, fmt.Sprintf("port port=\"%s\"", ic.DestinationPort))
			ruleParts = append(ruleParts, fmt.Sprintf("protocol=\"%s\"", ic.Layer4Protocol))
		} else {
			ruleParts = append(ruleParts, fmt.Sprintf("service name=\"%s\"", ic.DestinationPort))
		}
	}
	ruleParts = append(ruleParts, "family=\"ipv4\"", policy)
	rule = strings.Join(ruleParts, " ")
	if err := fwd.FwD1.Call(fwd1Interface+".zone.addRichRule", 0, fwd.Zone, rule, 0).Store(&callRet); err != nil {
		log.Printf("Adding rich rule %#v failed: %s", rule, err)
		return callRet, err
	}
	return callRet, nil
}

func (fwd *FirewallDControl) OpenC2Act(oc2cmd openc2.OpenC2Command) error {
	log.Printf("Command: %#v", oc2cmd)
	if target, ok := oc2cmd.Target.(*openc2.OpenC2IPConnection); ok {
		var policy string
		switch oc2cmd.Action {
		case "deny":
			policy = "reject"
		case "allow":
			policy = "accept"
		default:
			log.Printf("Don't know what to do with action %#v", oc2cmd.Action)
		}
		if res, err := fwd.AddIC2Rule(policy, target); err != nil {
			return err
		} else {
			log.Printf("Action done: %#v", res)
		}
	} else {
		log.Printf("Unknown target: %#v", oc2cmd.Target)
	}
	return nil
}

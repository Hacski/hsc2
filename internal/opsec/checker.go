package opsec

import (
	"net"
	"strings"
	"sync"
)

type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

type Warning struct {
	Code     string
	Severity Severity
	Message  string
}

type Checker struct {
	mu       sync.RWMutex
	iocList  map[string]bool
	sigList  map[string]bool
}

func NewChecker() *Checker {
	return &Checker{
		iocList: map[string]bool{},
		sigList: map[string]bool{},
	}
}

func (c *Checker) AddIOC(ioc string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.iocList[strings.ToLower(strings.TrimSpace(ioc))] = true
}

func (c *Checker) AddSignature(sig string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sigList[strings.ToLower(strings.TrimSpace(sig))] = true
}

func (c *Checker) CheckTarget(target string) []Warning {
	var warns []Warning
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := strings.ToLower(strings.TrimSpace(target))
	if c.iocList[key] {
		warns = append(warns, Warning{
			Code:     "IOC_FLAGGED",
			Severity: SeverityHigh,
			Message:  "target " + target + " is in the known IOC list",
		})
	}
	if host, _, err := net.SplitHostPort(target); err == nil {
		hostKey := strings.ToLower(host)
		if c.iocList[hostKey] {
			warns = append(warns, Warning{
				Code:     "IOC_HOST_FLAGGED",
				Severity: SeverityHigh,
				Message:  "host " + host + " is in the known IOC list",
			})
		}
	}
	if isPrivateTarget(key) {
		warns = append(warns, Warning{
			Code:     "PRIVATE_RANGE",
			Severity: SeverityLow,
			Message:  "target " + target + " resolves to a private/RFC1918 range",
		})
	}
	return warns
}

func (c *Checker) CheckPayload(payload []byte) []Warning {
	var warns []Warning
	c.mu.RLock()
	defer c.mu.RUnlock()
	lower := strings.ToLower(string(payload))
	for sig := range c.sigList {
		if strings.Contains(lower, sig) {
			warns = append(warns, Warning{
				Code:     "KNOWN_SIGNATURE",
				Severity: SeverityHigh,
				Message:  "payload contains known bad signature: " + sig,
			})
		}
	}
	if hasSuspiciousStrings(payload) {
		warns = append(warns, Warning{
			Code:     "SUSPICIOUS_STRINGS",
			Severity: SeverityMedium,
			Message:  "payload contains suspicious plaintext strings",
		})
	}
	return warns
}

func (c *Checker) CheckListenerPort(port int) []Warning {
	var warns []Warning
	wellKnown := map[int]string{
		80:   "HTTP",
		443:  "HTTPS",
		8080: "alt-HTTP",
		53:   "DNS",
		22:   "SSH",
	}
	if svc, ok := wellKnown[port]; ok {
		_ = svc
		return warns
	}
	if port < 1024 {
		warns = append(warns, Warning{
			Code:     "PRIVILEGED_PORT",
			Severity: SeverityMedium,
			Message:  "listener on privileged port requires root",
		})
	}
	return warns
}

func isPrivateTarget(target string) bool {
	host := target
	if h, _, err := net.SplitHostPort(target); err == nil {
		host = h
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
	}
	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func hasSuspiciousStrings(payload []byte) bool {
	indicators := []string{
		"cmd.exe",
		"powershell",
		"mimikatz",
		"meterpreter",
		"metasploit",
	}
	lower := strings.ToLower(string(payload))
	for _, ind := range indicators {
		if strings.Contains(lower, ind) {
			return true
		}
	}
	return false
}

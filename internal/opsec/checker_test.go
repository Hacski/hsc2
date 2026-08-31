package opsec

import (
	"testing"
)

func TestCheckTargetIOC(t *testing.T) {
	c := NewChecker()
	c.AddIOC("evil.example.com")
	c.AddIOC("203.0.113.1")

	warns := c.CheckTarget("evil.example.com")
	if len(warns) == 0 {
		t.Fatal("expected IOC warning for flagged target")
	}
	if warns[0].Code != "IOC_FLAGGED" {
		t.Fatalf("expected IOC_FLAGGED, got %s", warns[0].Code)
	}

	warns = c.CheckTarget("safe.example.com")
	for _, w := range warns {
		if w.Code == "IOC_FLAGGED" || w.Code == "IOC_HOST_FLAGGED" {
			t.Fatalf("unexpected IOC warning for safe target: %v", w)
		}
	}
}

func TestCheckTargetPrivateRange(t *testing.T) {
	c := NewChecker()
	warns := c.CheckTarget("192.168.1.100:80")
	found := false
	for _, w := range warns {
		if w.Code == "PRIVATE_RANGE" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected PRIVATE_RANGE warning for 192.168.x.x target")
	}
}

func TestCheckPayloadSignature(t *testing.T) {
	c := NewChecker()
	c.AddSignature("mimikatz")
	warns := c.CheckPayload([]byte("invoke-mimikatz sekurlsa::logonpasswords"))
	found := false
	for _, w := range warns {
		if w.Code == "KNOWN_SIGNATURE" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected KNOWN_SIGNATURE warning")
	}
}

func TestCheckPayloadSuspiciousStrings(t *testing.T) {
	c := NewChecker()
	warns := c.CheckPayload([]byte("C:\\Windows\\System32\\cmd.exe /c whoami"))
	found := false
	for _, w := range warns {
		if w.Code == "SUSPICIOUS_STRINGS" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected SUSPICIOUS_STRINGS warning")
	}
}

func TestCheckListenerPort(t *testing.T) {
	c := NewChecker()

	warns := c.CheckListenerPort(443)
	if len(warns) != 0 {
		t.Fatalf("expected no warnings for port 443, got %v", warns)
	}

	warns = c.CheckListenerPort(4444)
	for _, w := range warns {
		if w.Code == "PRIVILEGED_PORT" {
			t.Fatal("port 4444 should not be flagged as privileged")
		}
	}
}

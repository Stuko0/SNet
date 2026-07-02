package network

import (
	"testing"
)

func TestConnectivityStatusString(t *testing.T) {
	tests := []struct {
		status   ConnectivityStatus
		expected string
	}{
		{ConnectivityUnknown, "unknown"},
		{ConnectivityNone, "none"},
		{ConnectivityPortal, "portal"},
		{ConnectivityLimited, "limited"},
		{ConnectivityFull, "full"},
		{ConnectivityStatus(99), "unknown"},
	}

	for _, tc := range tests {
		if tc.status.String() != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, tc.status.String())
		}
	}
}

func TestWiFiNetworkSignalBars(t *testing.T) {
	tests := []struct {
		signal   int
		expected string
	}{
		{85, "████████"},
		{65, "██████░░"},
		{45, "████░░░░"},
		{25, "██░░░░░░"},
		{5, "█░░░░░░░"},
	}

	for _, tc := range tests {
		n := WiFiNetwork{Signal: tc.signal}
		if n.SignalBars() != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, n.SignalBars())
		}
	}
}

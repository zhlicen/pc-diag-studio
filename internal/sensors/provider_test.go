package sensors

import "testing"

func TestParseProviderOutputNormalizesReadings(t *testing.T) {
	raw := []byte(`{
		"provider":"test",
		"readings":[
			{"kind":"temp","name":"CPU Package","unit":"C","value":91.5,"source":"hwinfo"},
			{"kind":"rpm","name":"Fan 1","unit":"RPM","value":2800},
			{"kind":"temperature","name":"","unit":"C","value":10}
		]
	}`)
	snap, err := parseProviderOutput(raw)
	if err != nil {
		t.Fatalf("parseProviderOutput: %v", err)
	}
	if snap.Status != "" {
		t.Fatalf("parser should not set runtime status, got %q", snap.Status)
	}
	if len(snap.Readings) != 2 {
		t.Fatalf("readings=%d, want 2", len(snap.Readings))
	}
	if snap.Readings[0].Kind != "temperature" || snap.Readings[1].Kind != "fan" {
		t.Fatalf("unexpected normalized kinds: %#v", snap.Readings)
	}
	if snap.CapturedAt == "" {
		t.Fatal("CapturedAt should be filled")
	}
}

func TestParseProviderOutputRejectsEmptyReadings(t *testing.T) {
	if _, err := parseProviderOutput([]byte(`{"provider":"test","readings":[]}`)); err == nil {
		t.Fatal("expected empty readings to fail")
	}
}

func TestParseProviderOutputAllowsAbsentStatus(t *testing.T) {
	snap, err := parseProviderOutput([]byte(`{"provider":"test","status":"absent","detail":"not running"}`))
	if err != nil {
		t.Fatalf("parseProviderOutput: %v", err)
	}
	if snap.Status != "absent" || snap.Detail != "not running" {
		t.Fatalf("unexpected absent snapshot: %#v", snap)
	}
}

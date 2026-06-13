package sensors

import "testing"

func TestDCMTemperatureKeepsDellRawCelsius(t *testing.T) {
	got := dcmValue(53, -1, "temperature")
	if got != 53 {
		t.Fatalf("dcmValue(53, -1, temperature)=%v, want 53", got)
	}
}

func TestDCMVoltageUsesUnitModifier(t *testing.T) {
	got := dcmValue(661, -3, "voltage")
	if got < 0.660 || got > 0.662 {
		t.Fatalf("dcmValue(661, -3, voltage)=%v, want about 0.661", got)
	}
}

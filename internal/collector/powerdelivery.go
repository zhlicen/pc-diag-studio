package collector

import (
	"diagnostic-studio/internal/model"
)

// BatteryStatus from root\wmi reports live charge/discharge rate in mW.
// Discharging while PowerOnline is the measurable signature of an
// undersized or failing AC adapter.
const batteryStatusScript = `
$b = Get-CimInstance -Namespace root/wmi -ClassName BatteryStatus -ErrorAction SilentlyContinue | Select-Object -First 1 PowerOnline, Charging, Discharging, ChargeRate, DischargeRate
if ($null -eq $b) { '{}' } else { $b | ConvertTo-Json }
`

func readBatteryStatus(atSec int) (model.BatteryReading, bool) {
	var raw struct {
		PowerOnline   *bool `json:"PowerOnline"`
		Charging      *bool `json:"Charging"`
		Discharging   *bool `json:"Discharging"`
		ChargeRate    *int  `json:"ChargeRate"`
		DischargeRate *int  `json:"DischargeRate"`
	}
	if err := runPSJSON(batteryStatusScript, &raw); err != nil || raw.PowerOnline == nil {
		return model.BatteryReading{}, false
	}
	r := model.BatteryReading{AtSec: atSec, PowerOnline: *raw.PowerOnline}
	if raw.Charging != nil {
		r.Charging = *raw.Charging
	}
	if raw.Discharging != nil {
		r.Discharging = *raw.Discharging
	}
	if raw.ChargeRate != nil {
		r.ChargeRateMW = *raw.ChargeRate
	}
	if raw.DischargeRate != nil {
		r.DischargeRateMW = *raw.DischargeRate
	}
	return r, true
}

// summarizePowerDelivery folds readings into the AC-drain verdict.
func summarizePowerDelivery(readings []model.BatteryReading) model.PowerDelivery {
	pd := model.PowerDelivery{Readings: readings}
	for _, r := range readings {
		if r.DischargeRateMW > pd.MaxDischargeMW {
			pd.MaxDischargeMW = r.DischargeRateMW
		}
		if r.PowerOnline && r.Discharging && r.DischargeRateMW > 0 {
			pd.ACDrainDetected = true
		}
	}
	return pd
}

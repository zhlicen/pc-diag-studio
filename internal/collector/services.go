package collector

import (
	"strings"

	"diagnostic-studio/internal/model"
)

const servicesScript = `
$s = Get-CimInstance Win32_Service | Select-Object Name,DisplayName,State,StartMode,PathName
ConvertTo-Json -InputObject @($s) -Depth 2
`

type rawService struct {
	Name        string `json:"Name"`
	DisplayName string `json:"DisplayName"`
	State       string `json:"State"`
	StartMode   string `json:"StartMode"`
	PathName    string `json:"PathName"`
}

// vendorHint classifies a service. Power-manager hints (dell-optimizer,
// dell-power-manager, intel-dtt) are the ones that can constrain CPU
// frequency; the rest provide background-load context.
func vendorHint(s rawService) string {
	name := strings.ToLower(s.Name)
	display := strings.ToLower(s.DisplayName)
	both := name + " " + display

	switch {
	case strings.Contains(both, "dell optimizer") || strings.Contains(name, "delloptimizer"):
		return model.HintDellOptimizer
	case strings.Contains(both, "dell") && strings.Contains(both, "power manager"):
		return model.HintDellPowerManager
	case strings.Contains(both, "supportassist"):
		return model.HintDellSupportAsst
	case strings.Contains(both, "dell"):
		return model.HintDellOther
	case strings.Contains(both, "dynamic tuning") || strings.Contains(both, "dptf") ||
		name == "esifsvc" || strings.Contains(name, "dttservice") || strings.Contains(name, "ipfsvc") ||
		strings.Contains(both, "innovation platform framework"):
		return model.HintIntelDTT
	case strings.Contains(both, "intel"):
		return model.HintIntelOther
	}
	return ""
}

func collectVendorServices(notes *[]string) []model.ServiceInfo {
	var raw []rawService
	if err := runPSJSON(servicesScript, &raw); err != nil {
		*notes = append(*notes, "services: "+err.Error())
		return nil
	}
	var out []model.ServiceInfo
	for _, s := range raw {
		hint := vendorHint(s)
		if hint == "" {
			continue
		}
		out = append(out, model.ServiceInfo{
			Name:        s.Name,
			DisplayName: s.DisplayName,
			State:       s.State,
			StartMode:   s.StartMode,
			VendorHint:  hint,
			PathName:    s.PathName,
		})
	}
	return out
}

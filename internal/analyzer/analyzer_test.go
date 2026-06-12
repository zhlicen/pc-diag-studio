package analyzer

import (
	"testing"

	"diagnostic-studio/internal/model"
)

func TestVendorServicesLabel(t *testing.T) {
	services := []model.ServiceInfo{
		{Name: "ipfsvc", State: "Running", VendorHint: model.HintIntelDTT},
		{Name: "IntelDSA", State: "Running", VendorHint: model.HintIntelOther},
	}
	if got := vendorServicesLabel(services); got != "Intel" {
		t.Fatalf("vendorServicesLabel(Intel only) = %q, want Intel", got)
	}

	services = append(services, model.ServiceInfo{Name: "DellOptimizer", State: "Running", VendorHint: model.HintDellOptimizer})
	if got := vendorServicesLabel(services); got != "Dell/Intel" {
		t.Fatalf("vendorServicesLabel(Dell+Intel) = %q, want Dell/Intel", got)
	}
}

func TestVendorServicesFindingCarriesVendorLabel(t *testing.T) {
	r := model.DiagnosticReport{
		VendorServices: []model.ServiceInfo{
			{Name: "ipfsvc", State: "Running", VendorHint: model.HintIntelDTT},
			{Name: "IntelDSA", State: "Running", VendorHint: model.HintIntelOther},
			{Name: "IntelTelemetry", State: "Running", VendorHint: model.HintIntelOther},
			{Name: "IntelUpdater", State: "Running", VendorHint: model.HintIntelOther},
		},
	}

	Analyze(&r)

	var finding *model.Finding
	for i := range r.Analysis.Findings {
		if r.Analysis.Findings[i].RuleID == RuleVendorServices {
			finding = &r.Analysis.Findings[i]
			break
		}
	}
	if finding == nil {
		t.Fatal("expected vendor-services finding")
	}
	if got := finding.Params["vendorLabel"]; got != "Intel" {
		t.Fatalf("finding vendorLabel = %v, want Intel", got)
	}
	if got := finding.Evidence[0].Params["vendorLabel"]; got != "Intel" {
		t.Fatalf("evidence vendorLabel = %v, want Intel", got)
	}
}

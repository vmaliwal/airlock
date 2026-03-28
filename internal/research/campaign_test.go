package research

import "testing"

func TestValidateCampaignPlan(t *testing.T) {
	plan := CampaignPlan{
		Objective:    "run multiple issue contracts",
		ArtifactsDir: "/tmp/airlock-campaign",
		Entries: []CampaignEntry{
			{Name: "one", Contract: "examples/one.json"},
			{Name: "two", Contract: "examples/two.json"},
		},
	}
	if errs := ValidateCampaignPlan(plan); len(errs) != 0 {
		t.Fatalf("expected valid campaign plan, got %v", errs)
	}
}

func TestValidateCampaignPlanRequiresEntries(t *testing.T) {
	plan := CampaignPlan{Objective: "x", ArtifactsDir: "/tmp/y"}
	if errs := ValidateCampaignPlan(plan); len(errs) == 0 {
		t.Fatal("expected validation errors")
	}
}

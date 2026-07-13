package cron

import (
	"strings"
	"testing"
)

func TestFillAutomationBlueprint(t *testing.T) {
	job, err := FillAutomationBlueprint("news-digest", map[string]string{
		"topic":      "robotics",
		"time":       "09:15",
		"recurrence": "weekdays",
		"count":      "3",
		"deliver":    "local",
	})
	if err != nil {
		t.Fatalf("FillAutomationBlueprint() error = %v", err)
	}
	if job.Name != "Topic news digest" {
		t.Fatalf("Name = %q, want Topic news digest", job.Name)
	}
	if job.Schedule.Kind != "cron" || job.Schedule.Expr != "15 9 * * 1-5" {
		t.Fatalf("Schedule = %+v, want cron 15 9 * * 1-5", job.Schedule)
	}
	if !strings.Contains(job.Prompt, "robotics") || !strings.Contains(job.Prompt, "3 bullets") {
		t.Fatalf("Prompt did not fill values: %q", job.Prompt)
	}
	if job.Deliver {
		t.Fatal("Deliver = true, want false for local")
	}
}

func TestFillAutomationBlueprintRejectsInvalidEnum(t *testing.T) {
	_, err := FillAutomationBlueprint("meal-plan", map[string]string{"diet": "pizza-only"})
	if err == nil {
		t.Fatal("expected invalid enum error")
	}
}

func TestFillAutomationBlueprintOriginDelivery(t *testing.T) {
	job, err := FillAutomationBlueprint("morning-brief", map[string]string{"deliver": "origin"})
	if err != nil {
		t.Fatalf("FillAutomationBlueprint() error = %v", err)
	}
	if !job.Deliver {
		t.Fatal("Deliver = false, want true for origin")
	}
}

package conformance

import (
	"context"
	"testing"
)

func TestCatalogStatuses(t *testing.T) {
	cat := Catalog()
	if len(cat) < 20 {
		t.Fatalf("catalog too small: %d", len(cat))
	}
	for _, it := range cat {
		switch it.Status {
		case StatusImplemented, StatusPartial, StatusGap, StatusNA:
		default:
			t.Fatalf("bad status %s on %s", it.Status, it.ID)
		}
		if it.ID == "" || it.Spec == "" {
			t.Fatal("empty id/spec")
		}
	}
}

func TestProfilesReferenceItems(t *testing.T) {
	for _, p := range Profiles() {
		if len(p.ItemIDs) == 0 {
			t.Fatal(p.ID)
		}
		for _, id := range p.ItemIDs {
			if _, ok := ItemByID(id); !ok {
				t.Fatalf("profile %s unknown item %s", p.ID, id)
			}
		}
	}
}

func TestRunHermesCoreClaim(t *testing.T) {
	rep, err := Run(context.Background(), Options{Profile: ClaimProfile})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.ClaimOK {
		t.Fatal(Format(rep))
	}
	if rep.ChecksPassed < 10 {
		t.Fatalf("checks passed %d", rep.ChecksPassed)
	}
}

func TestRunFullProfileGreen(t *testing.T) {
	rep, err := Run(context.Background(), Options{Profile: "aesp.profile.hermes-agent-os"})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Gap != 0 {
		t.Fatalf("expected 0 gaps, got %d\n%s", rep.Gap, Format(rep))
	}
	if !rep.ClaimOK {
		t.Fatal(Format(rep))
	}
}

func TestFormat(t *testing.T) {
	rep, err := Run(context.Background(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := Format(rep)
	if len(s) < 200 {
		t.Fatal("short report")
	}
}

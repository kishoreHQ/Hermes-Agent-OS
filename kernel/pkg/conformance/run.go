package conformance

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Report is a full conformance run.
type Report struct {
	Implementation string        `json:"implementation"`
	SuiteVersion   string        `json:"suiteVersion"`
	ClaimProfile   string        `json:"claimProfile"`
	GeneratedAt    time.Time     `json:"generatedAt"`
	Catalog        []Item        `json:"catalog"`
	Checks         []CheckResult `json:"checks"`
	// Counts by catalog status
	Implemented int `json:"implemented"`
	Partial     int `json:"partial"`
	Gap         int `json:"gap"`
	// Check stats
	ChecksPassed int `json:"checksPassed"`
	ChecksFailed int `json:"checksFailed"`
	// Claim green if all claimed-profile items that have checks passed, and no claimed item is missing required checks
	ClaimOK bool     `json:"claimOk"`
	Notes   []string `json:"notes,omitempty"`
}

// Options for Run.
type Options struct {
	// Profile to claim (default ClaimProfile)
	Profile string
	// SkipChecks runs catalog-only enumeration
	SkipChecks bool
}

// Run executes catalog enumeration + runtime checks for the claimed profile.
func Run(ctx context.Context, opts Options) (*Report, error) {
	profileID := opts.Profile
	if profileID == "" {
		profileID = ClaimProfile
	}
	rep := &Report{
		Implementation: ImplementationVersion,
		SuiteVersion:   SuiteVersion,
		ClaimProfile:   profileID,
		GeneratedAt:    time.Now().UTC(),
		Catalog:        Catalog(),
	}
	for _, it := range rep.Catalog {
		switch it.Status {
		case StatusImplemented:
			rep.Implemented++
		case StatusPartial:
			rep.Partial++
		case StatusGap:
			rep.Gap++
		}
	}

	// Build set of check IDs required by claimed profile
	var claimItems []Item
	for _, p := range Profiles() {
		if p.ID != profileID {
			continue
		}
		for _, id := range p.ItemIDs {
			if it, ok := ItemByID(id); ok {
				claimItems = append(claimItems, it)
			}
		}
	}
	if len(claimItems) == 0 {
		return rep, fmt.Errorf("unknown profile %s", profileID)
	}

	if opts.SkipChecks {
		rep.ClaimOK = false
		rep.Notes = append(rep.Notes, "checks skipped")
		return rep, nil
	}

	k, err := bootKernel("conformance")
	if err != nil {
		return nil, err
	}
	checks := Checks()
	// Deduplicate check ids for claimed items that are implemented or partial
	need := map[string]bool{}
	for _, it := range claimItems {
		if it.CheckID == "" {
			continue
		}
		if it.Status == StatusGap {
			continue
		}
		need[it.CheckID] = true
	}
	ids := make([]string, 0, len(need))
	for id := range need {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		fn, ok := checks[id]
		if !ok {
			rep.Checks = append(rep.Checks, CheckResult{ID: id, OK: false, Error: "check not registered"})
			rep.ChecksFailed++
			continue
		}
		cr := fn(ctx, k)
		rep.Checks = append(rep.Checks, cr)
		if cr.OK {
			rep.ChecksPassed++
		} else {
			rep.ChecksFailed++
		}
	}

	// Claim OK: all checks green AND no claimed item is StatusGap for core claim
	// For hermes-core we only require implemented/partial items' checks pass
	rep.ClaimOK = rep.ChecksFailed == 0 && rep.ChecksPassed > 0
	if rep.ClaimOK {
		// Ensure claimed implemented items with checks actually passed
		for _, it := range claimItems {
			if it.Status == StatusImplemented && it.CheckID != "" {
				found := false
				for _, c := range rep.Checks {
					if c.ID == it.CheckID && c.OK {
						found = true
						break
					}
				}
				if !found {
					rep.ClaimOK = false
					rep.Notes = append(rep.Notes, "missing pass for "+it.ID)
				}
			}
		}
	}

	// Full profile also requires zero catalog gaps among claimed items
	if profileID == "aesp.profile.hermes-agent-os" {
		for _, it := range claimItems {
			if it.Status == StatusGap {
				rep.ClaimOK = false
				rep.Notes = append(rep.Notes, "profile incomplete: gap "+it.ID)
			}
			if it.Status == StatusPartial {
				// allow partial only if check still passes; note it
				rep.Notes = append(rep.Notes, "partial: "+it.ID)
			}
		}
	}

	return rep, nil
}

// Format human-readable report.
func Format(rep *Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "AESP Conformance — Hermes Agent OS\n")
	fmt.Fprintf(&b, "  implementation: %s\n", rep.Implementation)
	fmt.Fprintf(&b, "  suite:          %s\n", rep.SuiteVersion)
	fmt.Fprintf(&b, "  claim profile:  %s\n", rep.ClaimProfile)
	fmt.Fprintf(&b, "  generated:      %s\n", rep.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "\nCatalog: implemented=%d partial=%d gap=%d (total=%d)\n",
		rep.Implemented, rep.Partial, rep.Gap, len(rep.Catalog))
	fmt.Fprintf(&b, "\nCatalog items:\n")
	for _, it := range rep.Catalog {
		fmt.Fprintf(&b, "  [%s] %s (%s) %s\n", it.Status, it.ID, it.Spec, it.Title)
		if it.Notes != "" {
			fmt.Fprintf(&b, "         note: %s\n", it.Notes)
		}
	}
	fmt.Fprintf(&b, "\nExecutable checks: passed=%d failed=%d\n", rep.ChecksPassed, rep.ChecksFailed)
	for _, c := range rep.Checks {
		mark := "FAIL"
		if c.OK {
			mark = "PASS"
		}
		fmt.Fprintf(&b, "  [%s] %s", mark, c.ID)
		if c.OK && c.Detail != "" {
			fmt.Fprintf(&b, " — %s", c.Detail)
		}
		if !c.OK && c.Error != "" {
			fmt.Fprintf(&b, " — %s", c.Error)
		}
		fmt.Fprintf(&b, "\n")
	}
	for _, n := range rep.Notes {
		fmt.Fprintf(&b, "  note: %s\n", n)
	}
	if rep.ClaimOK {
		fmt.Fprintf(&b, "\nRESULT: PASS — claim %s green\n", rep.ClaimProfile)
	} else {
		fmt.Fprintf(&b, "\nRESULT: FAIL — claim %s not green\n", rep.ClaimProfile)
	}
	fmt.Fprintf(&b, "\nObjective claim rules (AESP CONFORMANCE.md):\n")
	fmt.Fprintf(&b, "  profile=%s suite=%s impl=%s\n", rep.ClaimProfile, rep.SuiteVersion, rep.Implementation)
	return b.String()
}

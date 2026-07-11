package evaluation

import (
	"context"
	"testing"
)

func TestDefaultSuite(t *testing.T) {
	rep, err := Run(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Failed > 0 {
		t.Fatal(Format(rep))
	}
	if rep.Passed < 4 {
		t.Fatalf("passed %d", rep.Passed)
	}
}

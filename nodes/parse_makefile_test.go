package nodes_test

import (
	"context"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestParseMakefile(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ParseMakefile(ctx, ax, &gen.MakefileInput{Content: kitchenSinkMakefile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Targets) != 9 {
		t.Errorf("len(Targets) = %d, want 9 (install/uninstall split from one rule header)", len(got.Targets))
	}
	if len(got.Variables) != 10 {
		t.Errorf("len(Variables) = %d, want 10", len(got.Variables))
	}
	if got.DefaultGoal != "app" || !got.DefaultGoalFound {
		t.Errorf("DefaultGoal = %q found=%v, want app/true", got.DefaultGoal, got.DefaultGoalFound)
	}
	if got.LineCount != 51 {
		t.Errorf("LineCount = %d, want 51", got.LineCount)
	}
	if len(got.PhonyTargets) != 2 {
		t.Errorf("PhonyTargets = %v, want [all clean]", got.PhonyTargets)
	}
}

func TestParseMakefile_EmptyInput(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ParseMakefile(ctx, ax, &gen.MakefileInput{Content: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Targets) != 0 || len(got.Variables) != 0 {
		t.Errorf("expected an empty result for empty input, got %+v", got)
	}
}

func TestParseMakefile_OversizedInput_ReturnsError(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	huge := make([]byte, 3*1024*1024)
	for i := range huge {
		huge[i] = 'x'
	}

	_, err := nodes.ParseMakefile(ctx, ax, &gen.MakefileInput{Content: string(huge)})
	if err == nil {
		t.Fatal("expected an error for oversized input, got nil")
	}
}

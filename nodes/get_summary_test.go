package nodes_test

import (
	"context"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestGetSummary(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.GetSummary(ctx, ax, &gen.MakefileInput{Content: kitchenSinkMakefile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.TargetCount != 9 {
		t.Errorf("TargetCount = %d, want 9 (install/uninstall split from one rule header)", got.TargetCount)
	}
	if got.VariableCount != 10 {
		t.Errorf("VariableCount = %d, want 10", got.VariableCount)
	}
	if got.PhonyCount != 2 {
		t.Errorf("PhonyCount = %d, want 2", got.PhonyCount)
	}
	if got.PatternRuleCount != 1 {
		t.Errorf("PatternRuleCount = %d, want 1", got.PatternRuleCount)
	}
	if got.DoubleColonCount != 2 {
		t.Errorf("DoubleColonCount = %d, want 2", got.DoubleColonCount)
	}
	if got.CommentCount != 4 {
		t.Errorf("CommentCount = %d, want 4", got.CommentCount)
	}
	if got.DirectiveCount != 6 {
		t.Errorf("DirectiveCount = %d, want 6", got.DirectiveCount)
	}
	if got.LineCount != 51 {
		t.Errorf("LineCount = %d, want 51", got.LineCount)
	}
}

func TestGetSummary_Empty(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.GetSummary(ctx, ax, &gen.MakefileInput{Content: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TargetCount != 0 || got.VariableCount != 0 || got.LineCount != 0 {
		t.Errorf("expected an all-zero summary for empty input, got %+v", got)
	}
}

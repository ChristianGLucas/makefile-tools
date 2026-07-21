package nodes_test

import (
	"context"
	"reflect"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestListPatternRules(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ListPatternRules(ctx, ax, &gen.MakefileInput{Content: kitchenSinkMakefile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.PatternRules) != 1 {
		t.Fatalf("expected 1 pattern rule, got %d: %+v", len(got.PatternRules), got.PatternRules)
	}
	if got.PatternRules[0].Name != "%.o" {
		t.Errorf("Name = %q, want %%.o", got.PatternRules[0].Name)
	}
	if !reflect.DeepEqual(got.PatternRules[0].Prerequisites, []string{"%.c"}) {
		t.Errorf("Prerequisites = %v, want [%%.c]", got.PatternRules[0].Prerequisites)
	}
}

func TestListPatternRules_None(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ListPatternRules(ctx, ax, &gen.MakefileInput{Content: "all:\n\t@echo hi\n"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.PatternRules) != 0 {
		t.Errorf("expected no pattern rules, got %+v", got.PatternRules)
	}
}

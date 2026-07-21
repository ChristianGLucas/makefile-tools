package nodes_test

import (
	"context"
	"reflect"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestListDoubleColonRules(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ListDoubleColonRules(ctx, ax, &gen.MakefileInput{Content: kitchenSinkMakefile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.DoubleColonRules) != 2 {
		t.Fatalf("expected 2 double-colon rules, got %d: %+v", len(got.DoubleColonRules), got.DoubleColonRules)
	}
	for _, r := range got.DoubleColonRules {
		if r.Name != "lib.a" || !r.IsDoubleColon {
			t.Errorf("unexpected double-colon rule: %+v", r)
		}
	}
}

func TestListDoubleColonRules_RealPrerequisites(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ListDoubleColonRules(ctx, ax, &gen.MakefileInput{Content: "lib.a:: a.o\n\t@echo a\nlib.a:: b.o\n\t@echo b\n"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.DoubleColonRules) != 2 {
		t.Fatalf("expected 2 rules, got %+v", got.DoubleColonRules)
	}
	// checkmake's tokenizer stops the target head at the first colon, which
	// left a spurious ":" as the first dependency of a "::" rule before this
	// package's fix; confirm it stays fixed.
	if !reflect.DeepEqual(got.DoubleColonRules[0].Prerequisites, []string{"a.o"}) {
		t.Errorf("Prerequisites[0] = %v, want [a.o] (no spurious leading \":\")", got.DoubleColonRules[0].Prerequisites)
	}
	if !reflect.DeepEqual(got.DoubleColonRules[1].Prerequisites, []string{"b.o"}) {
		t.Errorf("Prerequisites[1] = %v, want [b.o]", got.DoubleColonRules[1].Prerequisites)
	}
}

func TestListDoubleColonRules_None(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ListDoubleColonRules(ctx, ax, &gen.MakefileInput{Content: "all:\n\t@echo hi\n"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.DoubleColonRules) != 0 {
		t.Errorf("expected no double-colon rules, got %+v", got.DoubleColonRules)
	}
}

package nodes_test

import (
	"context"
	"reflect"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestListPhonyTargets(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ListPhonyTargets(ctx, ax, &gen.MakefileInput{Content: kitchenSinkMakefile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"all", "clean"}
	if !reflect.DeepEqual(got.PhonyTargets, want) {
		t.Errorf("PhonyTargets = %v, want %v", got.PhonyTargets, want)
	}
}

func TestListPhonyTargets_None(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ListPhonyTargets(ctx, ax, &gen.MakefileInput{Content: "all:\n\t@echo hi\n"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.PhonyTargets) != 0 {
		t.Errorf("expected no phony targets, got %v", got.PhonyTargets)
	}
}

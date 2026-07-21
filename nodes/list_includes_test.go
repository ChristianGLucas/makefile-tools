package nodes_test

import (
	"context"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestListIncludes(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ListIncludes(ctx, ax, &gen.MakefileInput{Content: kitchenSinkMakefile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Includes) != 2 {
		t.Fatalf("expected 2 includes, got %d: %+v", len(got.Includes), got.Includes)
	}
	if got.Includes[0].PathExpr != "config.mk" || got.Includes[0].Optional {
		t.Errorf("Includes[0] = %+v, want path=config.mk optional=false", got.Includes[0])
	}
	if got.Includes[1].PathExpr != "optional.mk" || !got.Includes[1].Optional {
		t.Errorf("Includes[1] = %+v, want path=optional.mk optional=true", got.Includes[1])
	}
}

func TestListIncludes_None(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ListIncludes(ctx, ax, &gen.MakefileInput{Content: "all:\n\t@echo hi\n"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Includes) != 0 {
		t.Errorf("expected no includes, got %+v", got.Includes)
	}
}

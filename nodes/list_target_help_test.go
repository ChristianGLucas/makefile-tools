package nodes_test

import (
	"context"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestListTargetHelp(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ListTargetHelp(ctx, ax, &gen.MakefileInput{Content: kitchenSinkMakefile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only "all", "long.o", and "clean" have associated help text in the
	// fixture; "app", "%.o", "install uninstall", and "lib.a" (x2) do not
	// and must be omitted.
	if len(got.Docs) != 3 {
		names := make([]string, len(got.Docs))
		for i, d := range got.Docs {
			names[i] = d.TargetName
		}
		t.Fatalf("expected 3 documented targets, got %d: %v", len(got.Docs), names)
	}
	byName := map[string][]string{}
	for _, d := range got.Docs {
		byName[d.TargetName] = d.HelpText
	}
	if byName["all"] == nil || byName["all"][0] != "Build everything" {
		t.Errorf("all's help text = %v", byName["all"])
	}
	if byName["long.o"] == nil {
		t.Errorf("expected long.o to have help text")
	}
	if byName["clean"] == nil {
		t.Errorf("expected clean to have help text")
	}
}

func TestListTargetHelp_NoDocumentedTargets(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ListTargetHelp(ctx, ax, &gen.MakefileInput{Content: "all:\n\t@echo hi\n"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Docs) != 0 {
		t.Errorf("expected no documented targets, got %+v", got.Docs)
	}
}

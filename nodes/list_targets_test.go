package nodes_test

import (
	"context"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestListTargets(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ListTargets(ctx, ax, &gen.MakefileInput{Content: kitchenSinkMakefile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Targets) != 9 {
		t.Fatalf("len(Targets) = %d, want 9", len(got.Targets))
	}

	var names []string
	for _, tg := range got.Targets {
		names = append(names, tg.Name)
	}
	// "install uninstall:" is one rule naming two targets — each gets its
	// own entry, not one bogus "install uninstall" name.
	want := []string{"all", "app", "%.o", "long.o", "install", "uninstall", "lib.a", "lib.a", "clean"}
	if len(names) != len(want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("Targets[%d].Name = %q, want %q (source order)", i, names[i], want[i])
		}
	}
}

func TestListTargets_NoTargets(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ListTargets(ctx, ax, &gen.MakefileInput{Content: "CC := gcc\n"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Targets) != 0 {
		t.Errorf("expected no targets for a variable-only Makefile, got %+v", got.Targets)
	}
}

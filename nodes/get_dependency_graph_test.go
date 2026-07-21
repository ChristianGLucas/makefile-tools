package nodes_test

import (
	"context"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestGetDependencyGraph_BuildOrder(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	content := "all: build test\nbuild:\n\t@echo build\ntest: build\n\t@echo test\n"
	got, err := nodes.GetDependencyGraph(ctx, ax, &gen.MakefileInput{Content: content})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.HasCycle {
		t.Fatalf("did not expect a cycle: %+v", got)
	}
	pos := map[string]int{}
	for i, n := range got.BuildOrder {
		pos[n] = i
	}
	if pos["build"] >= pos["test"] || pos["test"] >= pos["all"] {
		t.Errorf("build order %v violates dependency order (build < test < all)", got.BuildOrder)
	}
}

func TestGetDependencyGraph_CycleDetected(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	content := "a: b\n\t@echo a\nb: c\n\t@echo b\nc: a\n\t@echo c\n"
	got, err := nodes.GetDependencyGraph(ctx, ax, &gen.MakefileInput{Content: content})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.HasCycle {
		t.Fatalf("expected has_cycle=true, got %+v", got)
	}
	if len(got.BuildOrder) != 0 {
		t.Errorf("expected empty build_order on a cycle, got %v", got.BuildOrder)
	}
	if len(got.CycleTargets) != 3 {
		t.Errorf("expected 3 cycle_targets, got %v", got.CycleTargets)
	}
}

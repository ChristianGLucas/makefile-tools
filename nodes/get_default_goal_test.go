package nodes_test

import (
	"context"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestGetDefaultGoal_FromDefaultGoalVariable(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.GetDefaultGoal(ctx, ax, &gen.MakefileInput{Content: kitchenSinkMakefile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Found || got.TargetName != "app" || got.Source != "DEFAULT_GOAL variable" {
		t.Errorf("got %+v, want target=app found=true source=\"DEFAULT_GOAL variable\"", got)
	}
}

func TestGetDefaultGoal_FirstTargetFallback(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.GetDefaultGoal(ctx, ax, &gen.MakefileInput{Content: "build: prep\n\t@echo build\nprep:\n\t@echo prep\n"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Found || got.TargetName != "build" || got.Source != "first target" {
		t.Errorf("got %+v, want target=build found=true source=\"first target\"", got)
	}
}

func TestGetDefaultGoal_NoTargets(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.GetDefaultGoal(ctx, ax, &gen.MakefileInput{Content: "CC := gcc\n"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Found {
		t.Errorf("expected found=false when there are no targets, got %+v", got)
	}
}

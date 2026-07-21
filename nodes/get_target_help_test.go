package nodes_test

import (
	"context"
	"reflect"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestGetTargetHelp_PrecedingComment(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.GetTargetHelp(ctx, ax, &gen.TargetQuery{Content: kitchenSinkMakefile, TargetName: "all"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Found {
		t.Fatalf("expected found=true")
	}
	if !reflect.DeepEqual(got.HelpText, []string{"Build everything"}) {
		t.Errorf("HelpText = %v, want [Build everything]", got.HelpText)
	}
}

func TestGetTargetHelp_TrailingComment(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.GetTargetHelp(ctx, ax, &gen.TargetQuery{Content: kitchenSinkMakefile, TargetName: "clean"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got.HelpText, []string{"trailing comment, not help"}) {
		t.Errorf("HelpText = %v", got.HelpText)
	}
}

func TestGetTargetHelp_NoComment(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.GetTargetHelp(ctx, ax, &gen.TargetQuery{Content: kitchenSinkMakefile, TargetName: "app"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Found {
		t.Fatalf("expected found=true")
	}
	if len(got.HelpText) != 0 {
		t.Errorf("expected no help text for 'app', got %v", got.HelpText)
	}
}

func TestGetTargetHelp_NotFound(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.GetTargetHelp(ctx, ax, &gen.TargetQuery{Content: kitchenSinkMakefile, TargetName: "nope"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Found {
		t.Errorf("expected found=false")
	}
}

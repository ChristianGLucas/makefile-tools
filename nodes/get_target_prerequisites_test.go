package nodes_test

import (
	"context"
	"reflect"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestGetTargetPrerequisites_Found(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.GetTargetPrerequisites(ctx, ax, &gen.TargetQuery{Content: kitchenSinkMakefile, TargetName: "app"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Found {
		t.Fatalf("expected found=true for 'app'")
	}
	want := []string{"main.o", "util.o", "long.o"}
	if !reflect.DeepEqual(got.Prerequisites, want) {
		t.Errorf("Prerequisites = %v, want %v", got.Prerequisites, want)
	}
}

func TestGetTargetPrerequisites_NotFound(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.GetTargetPrerequisites(ctx, ax, &gen.TargetQuery{Content: kitchenSinkMakefile, TargetName: "does-not-exist"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Found {
		t.Errorf("expected found=false for a nonexistent target")
	}
	if len(got.Prerequisites) != 0 {
		t.Errorf("expected no prerequisites, got %v", got.Prerequisites)
	}
}

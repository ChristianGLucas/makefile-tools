package nodes_test

import (
	"context"
	"reflect"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestGetTargetRecipe_Found(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.GetTargetRecipe(ctx, ax, &gen.TargetQuery{Content: kitchenSinkMakefile, TargetName: "app"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Found {
		t.Fatalf("expected found=true for 'app'")
	}
	want := []string{"@echo linking $@", "$(CC) $(CFLAGS) -o app main.o util.o"}
	if !reflect.DeepEqual(got.Recipe, want) {
		t.Errorf("Recipe = %v, want %v (inline ';' recipe followed by tab-indented body)", got.Recipe, want)
	}
}

func TestGetTargetRecipe_NoRecipeLines(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.GetTargetRecipe(ctx, ax, &gen.TargetQuery{Content: kitchenSinkMakefile, TargetName: "all"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Found {
		t.Fatalf("expected found=true for 'all'")
	}
	if len(got.Recipe) != 0 {
		t.Errorf("expected 'all' to have no recipe lines, got %v", got.Recipe)
	}
}

func TestGetTargetRecipe_NotFound(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.GetTargetRecipe(ctx, ax, &gen.TargetQuery{Content: kitchenSinkMakefile, TargetName: "nope"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Found {
		t.Errorf("expected found=false")
	}
}

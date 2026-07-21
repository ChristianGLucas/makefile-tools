package nodes_test

import (
	"context"
	"reflect"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestExpandVariable_Resolves(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	content := "GREETING = hello, $(NAME)!\nNAME = world\n"
	got, err := nodes.ExpandVariable(ctx, ax, &gen.VariableQuery{Content: content, VariableName: "GREETING"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Found || !got.FullyResolved {
		t.Fatalf("got %+v, want found=true fully_resolved=true", got)
	}
	if got.ExpandedValue != "hello, world!" {
		t.Errorf("ExpandedValue = %q, want %q", got.ExpandedValue, "hello, world!")
	}
}

func TestExpandVariable_CycleDetected(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	content := "A = $(B)\nB = $(A)\n"
	got, err := nodes.ExpandVariable(ctx, ax, &gen.VariableQuery{Content: content, VariableName: "A"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.CycleDetected {
		t.Fatalf("expected cycle_detected=true, got %+v", got)
	}
	want := []string{"A", "B", "A"}
	if !reflect.DeepEqual(got.CyclePath, want) {
		t.Errorf("CyclePath = %v, want %v", got.CyclePath, want)
	}
}

func TestExpandVariable_NotFound(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ExpandVariable(ctx, ax, &gen.VariableQuery{Content: "A = 1\n", VariableName: "NOPE"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Found {
		t.Errorf("expected found=false for an undefined variable")
	}
}

func TestExpandVariable_UnresolvedReference(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ExpandVariable(ctx, ax, &gen.VariableQuery{Content: "X = $(MISSING)\n", VariableName: "X"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.FullyResolved {
		t.Errorf("expected fully_resolved=false")
	}
	if !reflect.DeepEqual(got.UnresolvedReferences, []string{"MISSING"}) {
		t.Errorf("UnresolvedReferences = %v", got.UnresolvedReferences)
	}
}

package nodes_test

import (
	"context"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestListVariables(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ListVariables(ctx, ax, &gen.MakefileInput{Content: kitchenSinkMakefile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Variables) != 10 {
		t.Fatalf("len(Variables) = %d, want 10", len(got.Variables))
	}

	byName := map[string][]*gen.Variable{}
	for _, v := range got.Variables {
		byName[v.Name] = append(byName[v.Name], v)
	}

	cases := []struct {
		name  string
		value string
		op    string
	}{
		{"CC", "gcc", ":="},
		{"CFLAGS", "-Wall -O2", "="},
		{"EXTRA_FLAGS", "-g", "?="},
		{"SHELL_VAR", "echo sh", "!="},
		{".DEFAULT_GOAL", "app", ":="},
	}
	for _, c := range cases {
		vs, ok := byName[c.name]
		if !ok || len(vs) != 1 {
			t.Fatalf("expected exactly one %q, got %v", c.name, vs)
		}
		if vs[0].Value != c.value || vs[0].Operator != c.op {
			t.Errorf("%s = %+v, want value=%q operator=%q", c.name, vs[0], c.value, c.op)
		}
	}

	if len(byName["SOURCES"]) != 2 {
		t.Errorf("expected two SOURCES assignments (+=), got %v", byName["SOURCES"])
	}
	for _, v := range byName["SOURCES"] {
		if v.Operator != "+=" {
			t.Errorf("SOURCES operator = %q, want +=", v.Operator)
		}
	}
}

func TestListVariables_NoVariables(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ListVariables(ctx, ax, &gen.MakefileInput{Content: "all:\n\t@echo hi\n"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Variables) != 0 {
		t.Errorf("expected no variables, got %v", got.Variables)
	}
}

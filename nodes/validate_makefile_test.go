package nodes_test

import (
	"context"
	"strings"
	"testing"

	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/nodes"
)

func TestValidateMakefile_Clean(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ValidateMakefile(ctx, ax, &gen.MakefileInput{Content: "all:\n\t@echo hi\n"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Valid {
		t.Errorf("expected valid=true for a well-formed Makefile, got issues: %+v", got.Issues)
	}
}

func TestValidateMakefile_SpaceIndentedRecipe_WarnsButStaysValid(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ValidateMakefile(ctx, ax, &gen.MakefileInput{Content: "build:\n    @echo hi\n"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, iss := range got.Issues {
		if iss.Severity == "warning" && strings.Contains(iss.Message, "literal tab") {
			found = true
			if iss.LineNumber != 2 {
				t.Errorf("LineNumber = %d, want 2", iss.LineNumber)
			}
		}
	}
	if !found {
		t.Errorf("expected a space-indented-recipe warning, got %+v", got.Issues)
	}
	// A warning alone must not make the Makefile invalid — only an "error"
	// severity issue does.
	if !got.Valid {
		t.Errorf("a warning-only result should still be valid=true")
	}
}

func TestValidateMakefile_UnterminatedDefine_Invalid(t *testing.T) {
	ctx := context.Background()
	ax := newTestContext(t)

	got, err := nodes.ValidateMakefile(ctx, ax, &gen.MakefileInput{Content: "define GREETING\necho hi\n"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Valid {
		t.Errorf("expected valid=false for an unterminated define block")
	}
	found := false
	for _, iss := range got.Issues {
		if iss.Severity == "error" && strings.Contains(iss.Message, "unterminated") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an 'unterminated define' error, got %+v", got.Issues)
	}
}

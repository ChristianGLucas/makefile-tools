package nodes

import (
	"context"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// GetTargetRecipe extracts one named target's recipe: its raw, unexpanded
// command lines in source order. found is false when no rule defines that
// target name. Recipe text is never executed or shell-parsed — it is
// returned exactly as written.
func GetTargetRecipe(ctx context.Context, ax axiom.Context, input *gen.TargetQuery) (*gen.TargetRecipe, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	t, found := findTarget(parsed.Targets, input.GetTargetName())
	out := &gen.TargetRecipe{TargetName: input.GetTargetName(), Found: found}
	if found {
		out.Recipe = t.Recipe
	}
	return out, nil
}

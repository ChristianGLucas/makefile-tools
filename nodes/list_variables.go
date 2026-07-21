package nodes

import (
	"context"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// ListVariables lists every variable assignment in a Makefile, in source
// order: name, raw unexpanded value, and the exact assignment operator used
// ("=", ":=", "?=", "+=", "!=", "::=", ":::=").
func ListVariables(ctx context.Context, ax axiom.Context, input *gen.MakefileInput) (*gen.VariableList, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	return &gen.VariableList{Variables: toGenVariables(parsed.Variables)}, nil
}

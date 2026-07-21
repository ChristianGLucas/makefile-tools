package nodes

import (
	"context"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// ExpandVariable resolves one variable's simple $(VAR)/${VAR} references
// against the file's own other assignments. Pure and static: no shell, no
// environment, no built-in Make functions or automatic variables ($@, $<,
// ...) are evaluated. A reference cycle (A=$(B), B=$(A)) is detected and
// reported via cycle_detected/cycle_path rather than looped.
func ExpandVariable(ctx context.Context, ax axiom.Context, input *gen.VariableQuery) (*gen.ExpandedVariable, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	r := mkfile.ExpandVariable(input.GetVariableName(), parsed.Variables)

	return &gen.ExpandedVariable{
		Name:                 input.GetVariableName(),
		Found:                r.Found,
		OriginalValue:        r.OriginalValue,
		ExpandedValue:        r.ExpandedValue,
		FullyResolved:        r.FullyResolved,
		UnresolvedReferences: r.UnresolvedReferences,
		CycleDetected:        r.CycleDetected,
		CyclePath:            r.CyclePath,
	}, nil
}

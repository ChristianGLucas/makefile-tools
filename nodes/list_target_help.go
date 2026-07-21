package nodes

import (
	"context"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// ListTargetHelp lists the documentation comments associated with every
// target that has any — trailing "## help text" and/or a directly-preceding
// comment block — in source order. Targets with no associated comment are
// omitted, matching the common "grep for ## and print" self-documenting
// Makefile convention.
func ListTargetHelp(ctx context.Context, ax axiom.Context, input *gen.MakefileInput) (*gen.TargetHelpList, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	var docs []*gen.TargetDoc
	for _, t := range parsed.Targets {
		if len(t.HelpComments) == 0 {
			continue
		}
		docs = append(docs, &gen.TargetDoc{TargetName: t.Name, HelpText: t.HelpComments})
	}

	return &gen.TargetHelpList{Docs: docs}, nil
}

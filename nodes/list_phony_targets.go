package nodes

import (
	"context"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// ListPhonyTargets lists every target name declared via one or more
// ".PHONY:" lines, deduplicated and sorted. Names may include targets with
// no rule of their own (a common defensive pattern), and are not filtered
// against the file's actual rule set.
func ListPhonyTargets(ctx context.Context, ax axiom.Context, input *gen.MakefileInput) (*gen.PhonyTargetList, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	return &gen.PhonyTargetList{PhonyTargets: parsed.PhonyTargets}, nil
}

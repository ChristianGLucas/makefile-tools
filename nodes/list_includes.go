package nodes

import (
	"context"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// ListIncludes lists every "include" / "-include" / "sinclude" directive, in
// source order, with its raw (unexpanded, un-globbed) path expression and
// whether it is the "optional" form ("-include"/"sinclude": a missing file
// is silently ignored rather than an error). Directives are reported only —
// never fetched, globbed, or read; ListIncludes never touches the
// filesystem or network.
func ListIncludes(ctx context.Context, ax axiom.Context, input *gen.MakefileInput) (*gen.IncludeList, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	var includes []*gen.IncludeDirective
	for _, d := range parsed.Directives {
		switch d.Type {
		case "include":
			includes = append(includes, &gen.IncludeDirective{PathExpr: d.Arguments, Optional: false, LineNumber: int32(d.LineNumber)})
		case "-include", "sinclude":
			includes = append(includes, &gen.IncludeDirective{PathExpr: d.Arguments, Optional: true, LineNumber: int32(d.LineNumber)})
		}
	}

	return &gen.IncludeList{Includes: includes}, nil
}

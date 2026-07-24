package nodes

import (
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// checkBounds passes mkfile.Parse's error through unchanged. Parse itself no
// longer enforces any input-size bound (the platform owns that), so this is
// currently always nil, but nodes keep calling it so a future internal-parser
// failure still surfaces as a structured error rather than silently ignored.
func checkBounds(err error) error {
	return err
}

func toGenTarget(t mkfile.Target) *gen.Target {
	return &gen.Target{
		Name:          t.Name,
		Prerequisites: t.Prerequisites,
		Recipe:        t.Recipe,
		IsPhony:       t.IsPhony,
		IsDoubleColon: t.IsDoubleColon,
		IsPatternRule: t.IsPatternRule,
		LineNumber:    int32(t.LineNumber),
		HelpComments:  t.HelpComments,
	}
}

func toGenTargets(ts []mkfile.Target) []*gen.Target {
	out := make([]*gen.Target, 0, len(ts))
	for _, t := range ts {
		out = append(out, toGenTarget(t))
	}
	return out
}

func toGenVariable(v mkfile.Variable) *gen.Variable {
	return &gen.Variable{
		Name:       v.Name,
		Value:      v.Value,
		Operator:   v.Operator,
		LineNumber: int32(v.LineNumber),
	}
}

func toGenVariables(vs []mkfile.Variable) []*gen.Variable {
	out := make([]*gen.Variable, 0, len(vs))
	for _, v := range vs {
		out = append(out, toGenVariable(v))
	}
	return out
}

func toGenComment(c mkfile.Comment) *gen.Comment {
	return &gen.Comment{
		Text:         c.Text,
		LineNumber:   int32(c.LineNumber),
		IsHelpMarker: c.IsHelpMarker,
	}
}

func toGenComments(cs []mkfile.Comment) []*gen.Comment {
	out := make([]*gen.Comment, 0, len(cs))
	for _, c := range cs {
		out = append(out, toGenComment(c))
	}
	return out
}

func toGenDirective(d mkfile.Directive) *gen.Directive {
	return &gen.Directive{
		DirectiveType: d.Type,
		Arguments:     d.Arguments,
		LineNumber:    int32(d.LineNumber),
	}
}

func toGenDirectives(ds []mkfile.Directive) []*gen.Directive {
	out := make([]*gen.Directive, 0, len(ds))
	for _, d := range ds {
		out = append(out, toGenDirective(d))
	}
	return out
}

func toGenIssue(i mkfile.Issue) *gen.ValidationIssue {
	return &gen.ValidationIssue{
		Severity:   i.Severity,
		Message:    i.Message,
		LineNumber: int32(i.LineNumber),
	}
}

func toGenIssues(is []mkfile.Issue) []*gen.ValidationIssue {
	out := make([]*gen.ValidationIssue, 0, len(is))
	for _, i := range is {
		out = append(out, toGenIssue(i))
	}
	return out
}

func findTarget(targets []mkfile.Target, name string) (mkfile.Target, bool) {
	for _, t := range targets {
		if t.Name == name {
			return t, true
		}
	}
	return mkfile.Target{}, false
}

// Package mkfile is the shared, pure, static parsing core for every node in
// this package. It never executes a recipe, never shells out to `make`,
// never touches the network, and never reads a caller-supplied path — the
// only filesystem use is a single self-created, immediately-removed temp
// file used to hand pre-sanitized text to the underlying tokenizer.
//
// Tokenizing the hard, ambiguous part of Makefile syntax — telling a rule
// header apart from a variable assignment, and collecting a rule's recipe
// body (including the "target: deps ; recipe" inline form) — is delegated
// to github.com/checkmake/checkmake/parser (MIT), the hand-rolled tokenizer
// behind the checkmake Makefile linter. This package supplies everything
// checkmake does not: backslash-continuation joining, comment and directive
// extraction, the exact assignment operator, double-colon and pattern-rule
// flags, .DEFAULT_GOAL (a variable despite its dot-prefixed name — see the
// note below), variable expansion with cycle detection, dependency-graph /
// build-order derivation, and structural validation.
package mkfile

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	ckparser "github.com/checkmake/checkmake/parser"
)

// Input-surface bounds. Enforced before any parsing work happens.
const (
	MaxContentBytes = 2 * 1024 * 1024 // 2 MiB
	MaxLines        = 50000
	// MaxExpandDepth bounds variable-expansion recursion even for long
	// non-cyclic reference chains, independent of cycle detection.
	MaxExpandDepth = 64
)

// ErrTooLarge is returned when the caller-supplied content exceeds the
// bounds above. Nodes translate this into a structured, non-crashing result.
type ErrTooLarge struct {
	Reason string
}

func (e ErrTooLarge) Error() string { return e.Reason }

// ---- result types (node code maps these 1:1 onto proto messages) ----

type Target struct {
	Name          string
	Prerequisites []string
	Recipe        []string
	IsPhony       bool
	IsDoubleColon bool
	IsPatternRule bool
	LineNumber    int
	HelpComments  []string
}

type Variable struct {
	Name       string
	Value      string
	Operator   string
	LineNumber int
}

type Comment struct {
	Text         string
	LineNumber   int
	IsHelpMarker bool
}

type Directive struct {
	Type       string
	Arguments  string
	LineNumber int
}

type Issue struct {
	Severity   string
	Message    string
	LineNumber int
}

type Parsed struct {
	Targets          []Target
	Variables        []Variable
	Comments         []Comment
	Directives       []Directive
	PhonyTargets     []string
	DefaultGoal      string
	DefaultGoalFound bool
	DefaultGoalSource string
	Issues           []Issue
	LineCount        int
}

var (
	// Directive keywords reported verbatim, never evaluated.
	reDirective = regexp.MustCompile(`^(ifeq|ifneq|ifdef|ifndef|else|endif|define|endef|vpath|export|unexport|override|-include|sinclude|include)(?:\s+(.*))?$`)

	// .DEFAULT_GOAL is, despite its dot-prefixed target-shaped name, an
	// ordinary GNU Make *variable* ("`.DEFAULT_GOAL := run`"), not a rule.
	reDefaultGoalVar = regexp.MustCompile(`^\.DEFAULT_GOAL\s*(:::=|::=|:=|\?=|\+=|!=|=)\s*(.*)$`)

	// Exact assignment operator, re-derived independently of checkmake
	// (which collapses all of these into a single bool). Longest operators
	// first so e.g. ":::=" isn't cut short at ":=".
	reVarOperator = regexp.MustCompile(`^[A-Za-z0-9_.-]+\s*(:::=|::=|:=|\?=|\+=|!=|=)`)

	// reAssignRemainder tests whether text is itself a variable assignment —
	// used to detect "export VAR := value" / "override VAR OP value" (GNU
	// Make manual §6.7's own example), where "export"/"override" is a
	// modifier prefixing a real assignment rather than a bare name-list.
	reAssignRemainder = regexp.MustCompile(`^[A-Za-z0-9_.-]+\s*(:::=|::=|:=|\?=|\+=|!=|=)\s*.*$`)

	// Double-colon detection on an already-identified rule header line.
	reDoubleColon = regexp.MustCompile(`^[^:]+::(?:[^=]|$)`)
)

// Parse runs the full pipeline over raw Makefile text and returns a
// structured, best-effort result. It never panics and never returns a nil
// Parsed — a malformed or oversized input comes back as issues, not an error
// from this function (callers that want a hard bounds error can check for
// ErrTooLarge on the returned error).
func Parse(content string) (Parsed, error) {
	var result Parsed

	if len(content) > MaxContentBytes {
		return result, ErrTooLarge{Reason: fmt.Sprintf("content is %d bytes, exceeds the %d byte limit", len(content), MaxContentBytes)}
	}

	rawLines := splitLines(content)
	if len(rawLines) > MaxLines {
		return result, ErrTooLarge{Reason: fmt.Sprintf("content has %d lines, exceeds the %d line limit", len(rawLines), MaxLines)}
	}
	result.LineCount = len(rawLines)

	sanitized, comments, directives, defaultGoalVar, preIssues := preprocess(rawLines)
	result.Comments = comments
	result.Directives = directives
	result.Issues = append(result.Issues, preIssues...)

	ck, err := runCheckmake(sanitized)
	if err != nil {
		result.Issues = append(result.Issues, Issue{
			Severity: "error",
			Message:  "internal parser failure: " + err.Error(),
		})
		return result, nil
	}

	targets, phony := buildTargets(ck, sanitized)
	variables := buildVariables(ck, sanitized)
	if defaultGoalVar != nil {
		variables = append(variables, *defaultGoalVar)
		sortVariablesByLine(variables)
	}

	attachHelpComments(targets, comments)

	result.Targets = targets
	result.Variables = variables
	result.PhonyTargets = phony

	goal, found, source := computeDefaultGoal(variables, targets)
	result.DefaultGoal = goal
	result.DefaultGoalFound = found
	result.DefaultGoalSource = source

	result.Issues = append(result.Issues, validate(rawLines, result)...)

	return result, nil
}

func splitLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	if content == "" {
		return nil
	}
	lines := strings.Split(content, "\n")
	// A trailing newline produces one trailing empty element; drop it so
	// line counts match what an editor would show.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// preprocess walks the raw physical lines once and produces:
//   - sanitized: exactly len(rawLines) entries, ready to hand to checkmake
//     (backslash-continuations joined onto their first physical line with
//     the remainder blanked; comments, directives, and .DEFAULT_GOAL lines
//     blanked and reported separately). Line-number alignment with the
//     original source is preserved throughout.
//   - comments, directives: extracted, in source order.
//   - defaultGoalVar: non-nil if a ".DEFAULT_GOAL" assignment was found.
func preprocess(rawLines []string) (sanitized []string, comments []Comment, directives []Directive, defaultGoalVar *Variable, issues []Issue) {
	n := len(rawLines)
	sanitized = make([]string, n)

	inDefine := false
	defineStartLine := 0
	defineName := ""

	i := 0
	for i < n {
		line := rawLines[i]

		// Recipe lines (leading tab) are never touched: joined nowhere,
		// stripped of nothing, passed straight through so checkmake's own
		// recipe-collection logic sees exactly what the caller wrote.
		if strings.HasPrefix(line, "\t") {
			if inDefine {
				sanitized[i] = "" // inside a define/endef body: opaque, not a recipe
			} else {
				sanitized[i] = line
			}
			i++
			continue
		}

		trimmed := strings.TrimSpace(line)

		if inDefine {
			sanitized[i] = ""
			if trimmed == "endef" || strings.HasPrefix(trimmed, "endef ") {
				directives = append(directives, Directive{Type: "define", Arguments: defineName, LineNumber: defineStartLine})
				inDefine = false
			}
			i++
			continue
		}

		if trimmed == "" {
			sanitized[i] = ""
			i++
			continue
		}

		// Full-line comment.
		if strings.HasPrefix(trimmed, "#") {
			text := strings.TrimPrefix(trimmed, "#")
			isHelp := strings.HasPrefix(text, "#")
			text = strings.TrimSpace(strings.TrimPrefix(text, "#"))
			comments = append(comments, Comment{Text: text, LineNumber: i + 1, IsHelpMarker: isHelp})
			sanitized[i] = ""
			i++
			continue
		}

		// Join backslash-continuations (outside recipe context) before any
		// further classification, per GNU Make's own logical-line joining.
		joined, consumed := joinContinuation(rawLines, i)
		startLine := i + 1

		if m := reDirective.FindStringSubmatch(strings.TrimSpace(joined)); m != nil {
			directiveType := m[1]
			args := strings.TrimSpace(m[2])
			if directiveType == "define" {
				inDefine = true
				defineStartLine = startLine
				defineName = args
				for k := 0; k < consumed; k++ {
					sanitized[i+k] = ""
				}
				i += consumed
				continue
			}
			directives = append(directives, Directive{Type: directiveType, Arguments: args, LineNumber: startLine})
			// "export"/"override" are modifier keywords that can PREFIX a real
			// variable assignment ("export PATH := ...", "override CFLAGS +=
			// -Wall" — the latter is the GNU Make manual's own §6.7 example),
			// not just a bare "export VAR1 VAR2" name-list. When the remainder
			// after the keyword is itself an assignment, still record the
			// directive above for traceability, but ALSO let checkmake parse
			// the assignment normally by feeding it just the remainder.
			if (directiveType == "export" || directiveType == "override") && reAssignRemainder.MatchString(args) {
				sanitized[i] = args
				for k := 1; k < consumed; k++ {
					sanitized[i+k] = ""
				}
				i += consumed
				continue
			}
			for k := 0; k < consumed; k++ {
				sanitized[i+k] = ""
			}
			i += consumed
			continue
		}

		// .DEFAULT_GOAL is a variable, not a rule — see reDefaultGoalVar's
		// doc comment. Handle it directly so checkmake's dot-prefixed
		// special-target branch (which only understands rule syntax)
		// never sees it and misparses the "=" as dependency text.
		if m := reDefaultGoalVar.FindStringSubmatch(joined); m != nil {
			defaultGoalVar = &Variable{
				Name:       ".DEFAULT_GOAL",
				Operator:   m[1],
				Value:      strings.TrimSpace(m[2]),
				LineNumber: startLine,
			}
			for k := 0; k < consumed; k++ {
				sanitized[i+k] = ""
			}
			i += consumed
			continue
		}

		// Strip a trailing, unescaped "#" comment (make's own rule: text
		// from an unescaped "#" to end of line is a comment on any
		// non-recipe line). "\#" is unescaped to a literal "#".
		kept, trailing, hasTrailing := stripTrailingComment(joined)
		if hasTrailing {
			text := strings.TrimSpace(trailing)
			isHelp := strings.HasPrefix(text, "#")
			text = strings.TrimSpace(strings.TrimPrefix(text, "#"))
			comments = append(comments, Comment{Text: text, LineNumber: startLine, IsHelpMarker: isHelp})
		}

		sanitized[i] = strings.TrimRight(kept, " \t")
		for k := 1; k < consumed; k++ {
			sanitized[i+k] = ""
		}
		i += consumed
	}

	if inDefine {
		issues = append(issues, Issue{Severity: "error", Message: fmt.Sprintf("unterminated 'define %s' block (no matching 'endef')", defineName), LineNumber: defineStartLine})
	}

	return sanitized, comments, directives, defaultGoalVar, issues
}

// joinContinuation joins physical lines starting at rawLines[i] while each
// ends in an odd number of trailing backslashes (an escaped final backslash
// — "\\\\" — does not continue the line). Returns the joined logical text
// (backslash-newline replaced by a single space, matching GNU Make) and how
// many physical lines were consumed (always >= 1).
func joinContinuation(rawLines []string, i int) (string, int) {
	var b strings.Builder
	consumed := 0
	for i+consumed < len(rawLines) {
		line := rawLines[i+consumed]
		consumed++
		trailingBackslashes := 0
		for k := len(line) - 1; k >= 0 && line[k] == '\\'; k-- {
			trailingBackslashes++
		}
		continues := trailingBackslashes%2 == 1
		if continues {
			line = line[:len(line)-1]
		}
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(strings.TrimRight(line, " \t"))
		if !continues || i+consumed >= len(rawLines) {
			break
		}
	}
	return b.String(), consumed
}

// stripTrailingComment removes the first unescaped "#" and everything after
// it. "\#" unescapes to a literal "#" in the kept portion.
func stripTrailingComment(line string) (kept string, comment string, has bool) {
	var out strings.Builder
	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if c == '\\' && i+1 < len(runes) && runes[i+1] == '#' {
			out.WriteRune('#')
			i++
			continue
		}
		if c == '#' {
			return out.String(), string(runes[i+1:]), true
		}
		out.WriteRune(c)
	}
	return out.String(), "", false
}

func runCheckmake(sanitized []string) (ckparser.Makefile, error) {
	f, err := os.CreateTemp("", "axiom-makefile-tools-*.mk")
	if err != nil {
		return ckparser.Makefile{}, err
	}
	path := f.Name()
	defer func() {
		f.Close()
		os.Remove(path)
	}()

	content := strings.Join(sanitized, "\n")
	if _, err := f.WriteString(content); err != nil {
		return ckparser.Makefile{}, err
	}
	if err := f.Close(); err != nil {
		return ckparser.Makefile{}, err
	}

	return ckparser.Parse(filepath.Clean(path))
}

func buildTargets(ck ckparser.Makefile, sanitized []string) ([]Target, []string) {
	var targets []Target
	var phony []string
	phonySeen := map[string]bool{}

	for _, r := range ck.Rules {
		if r.Target == ".PHONY" {
			for _, d := range r.Dependencies {
				if !phonySeen[d] {
					phonySeen[d] = true
					phony = append(phony, d)
				}
			}
			continue
		}
		if strings.HasPrefix(r.Target, ".") {
			// Other special targets (.SUFFIXES, .DELETE_ON_ERROR, ...) are
			// real make directives, not build targets a caller composes a
			// dependency graph over — keep them out of the target list.
			continue
		}

		line := ""
		if r.LineNumber-1 >= 0 && r.LineNumber-1 < len(sanitized) {
			line = sanitized[r.LineNumber-1]
		}

		isDouble := reDoubleColon.MatchString(line)
		deps := r.Dependencies
		if isDouble {
			// checkmake's rule regex stops the target head before the
			// FIRST colon, so for a "target:: deps" line its dependency
			// text starts with the second colon and strings.Fields() turns
			// that leading ":" into a spurious first dependency element.
			// Verified against checkmake v0.3.2 directly (see mkfile_test.go).
			deps = stripLeadingColon(deps)
		}

		// A rule header may name MULTIPLE targets sharing one prerequisite
		// list and recipe ("install uninstall: prep" defines two separate
		// targets, each depending on prep) — GNU Make manual §4.2. Split on
		// whitespace rather than keeping the joined string as one bogus
		// target name (the common case, one name, splits into a slice of 1
		// and is unaffected).
		for _, name := range strings.Fields(r.Target) {
			targets = append(targets, Target{
				Name:          name,
				Prerequisites: deps,
				Recipe:        r.Body,
				IsDoubleColon: isDouble,
				IsPatternRule: strings.Contains(name, "%"),
				LineNumber:    r.LineNumber,
			})
		}
	}

	for i := range targets {
		if phonySeen[targets[i].Name] {
			targets[i].IsPhony = true
		}
	}

	sort.Strings(phony)
	return targets, phony
}

// stripLeadingColon drops a spurious leading ":" element from a double-colon
// rule's dependency list (see the comment at its one call site).
func stripLeadingColon(deps []string) []string {
	if len(deps) > 0 && deps[0] == ":" {
		return deps[1:]
	}
	return deps
}

func buildVariables(ck ckparser.Makefile, sanitized []string) []Variable {
	var out []Variable
	for _, v := range ck.Variables {
		// checkmake v0.3.2 has a verified off-by-one: parser.Variable.LineNumber
		// (and the dot-prefixed special-target branch) reports one line past
		// the true source line — unlike its ordinary Rule.LineNumber, which is
		// correct. Confirmed against checkmake's own scanner bootstrap (the
		// first loop iteration performs a "free" Scan() before any line is
		// classified, so LineNumber is one Scan() ahead of Text() everywhere
		// except where parseRuleOrVariable explicitly subtracts 1 for rules).
		trueLine := v.LineNumber - 1
		op := "="
		line := ""
		if trueLine-1 >= 0 && trueLine-1 < len(sanitized) {
			line = sanitized[trueLine-1]
		}
		if m := reVarOperator.FindStringSubmatch(line); m != nil {
			op = m[1]
		} else if v.SimplyExpanded {
			op = ":="
		}
		out = append(out, Variable{
			Name:       v.Name,
			Value:      v.Assignment,
			Operator:   op,
			LineNumber: trueLine,
		})
	}
	return out
}

func sortVariablesByLine(vs []Variable) {
	sort.SliceStable(vs, func(i, j int) bool { return vs[i].LineNumber < vs[j].LineNumber })
}

// attachHelpComments attaches, to each target: any comment on the target's
// own line (the common "target: deps ## help text" convention) plus any
// comment block directly above it with no blank line in between (the
// doc-comment convention), in source order.
func attachHelpComments(targets []Target, comments []Comment) {
	byLine := map[int][]string{}
	for _, c := range comments {
		byLine[c.LineNumber] = append(byLine[c.LineNumber], c.Text)
	}

	for i := range targets {
		var help []string

		// Walk upward through directly-preceding comment-only lines.
		var preceding []string
		line := targets[i].LineNumber - 1
		for {
			texts, ok := byLine[line]
			if !ok {
				break
			}
			preceding = append([]string{}, append(texts, preceding...)...)
			line--
		}
		help = append(help, preceding...)

		// Same-line trailing comment.
		if texts, ok := byLine[targets[i].LineNumber]; ok {
			help = append(help, texts...)
		}

		targets[i].HelpComments = help
	}
}

func computeDefaultGoal(variables []Variable, targets []Target) (string, bool, string) {
	for _, v := range variables {
		if v.Name == ".DEFAULT_GOAL" {
			if v.Value != "" {
				return v.Value, true, "DEFAULT_GOAL variable"
			}
			return "", false, ""
		}
	}
	for _, t := range targets {
		if strings.HasPrefix(t.Name, ".") || t.IsPatternRule {
			continue
		}
		return t.Name, true, "first target"
	}
	return "", false, ""
}

// ---- variable expansion ----

var reVarRef = regexp.MustCompile(`\$[({]([A-Za-z0-9_.-]+)[)}]`)

// ExpandResult is the outcome of expanding one variable's value.
type ExpandResult struct {
	Found                bool
	OriginalValue        string
	ExpandedValue        string
	FullyResolved        bool
	UnresolvedReferences []string
	CycleDetected        bool
	CyclePath            []string
}

// ExpandVariable resolves simple $(VAR)/${VAR} references in name's value
// against the other assignments in the same file. Pure and static: no
// shell, no environment, no built-in functions or automatic variables. Self-
// or mutually-referential chains are detected and reported, never looped.
func ExpandVariable(name string, variables []Variable) ExpandResult {
	byName := map[string]string{}
	for _, v := range variables {
		// Last assignment wins, matching GNU Make's own "last wins" for
		// unconditional operators; ?= (conditional) still overwrites here
		// since this parser does not model prior-definedness order beyond
		// source order, which is a reasonable static approximation.
		byName[v.Name] = v.Value
	}

	original, found := byName[name]
	if !found {
		return ExpandResult{Found: false}
	}

	var unresolved []string
	unresolvedSeen := map[string]bool{}
	visiting := map[string]bool{}
	var cyclePath []string
	cycleFound := false

	var expand func(value string, depth int, path []string) string
	expand = func(value string, depth int, path []string) string {
		if cycleFound {
			return value
		}
		if depth > MaxExpandDepth {
			return value
		}
		return reVarRef.ReplaceAllStringFunc(value, func(match string) string {
			if cycleFound {
				return match
			}
			ref := reVarRef.FindStringSubmatch(match)[1]
			if visiting[ref] {
				cycleFound = true
				cyclePath = append(append([]string{}, path...), ref)
				return match
			}
			refVal, ok := byName[ref]
			if !ok {
				if !unresolvedSeen[ref] {
					unresolvedSeen[ref] = true
					unresolved = append(unresolved, ref)
				}
				return match
			}
			visiting[ref] = true
			result := expand(refVal, depth+1, append(path, ref))
			delete(visiting, ref)
			return result
		})
	}

	visiting[name] = true
	expandedValue := expand(original, 0, []string{name})
	delete(visiting, name)

	if cycleFound {
		return ExpandResult{
			Found:         true,
			OriginalValue: original,
			ExpandedValue: original,
			FullyResolved: false,
			CycleDetected: true,
			CyclePath:     cyclePath,
		}
	}

	sort.Strings(unresolved)
	return ExpandResult{
		Found:                true,
		OriginalValue:        original,
		ExpandedValue:        expandedValue,
		FullyResolved:        len(unresolved) == 0,
		UnresolvedReferences: unresolved,
	}
}

// ---- dependency graph ----

type DependencyEdge struct {
	Target           string
	DependsOnTargets []string
}

type DependencyGraphResult struct {
	Edges        []DependencyEdge
	BuildOrder   []string
	HasCycle     bool
	CycleTargets []string
}

// BuildDependencyGraph reduces each target's prerequisites to only those
// that are themselves targets defined in the file (a prerequisite that is
// just a source file has no further build step to sequence) and computes a
// topological build order, detecting cycles rather than looping forever.
func BuildDependencyGraph(targets []Target) DependencyGraphResult {
	known := map[string]bool{}
	for _, t := range targets {
		known[t.Name] = true
	}

	adj := map[string][]string{}
	order := make([]string, 0, len(targets))
	seenTarget := map[string]bool{}
	for _, t := range targets {
		if seenTarget[t.Name] {
			continue // duplicate rule for the same target: edges already recorded
		}
		seenTarget[t.Name] = true
		order = append(order, t.Name)
		var deps []string
		for _, p := range t.Prerequisites {
			if known[p] {
				deps = append(deps, p)
			}
		}
		adj[t.Name] = deps
	}

	var edges []DependencyEdge
	for _, name := range order {
		edges = append(edges, DependencyEdge{Target: name, DependsOnTargets: adj[name]})
	}

	buildOrder, cycleNodes, hasCycle := topoSort(order, adj)

	return DependencyGraphResult{
		Edges:        edges,
		BuildOrder:   buildOrder,
		HasCycle:     hasCycle,
		CycleTargets: cycleNodes,
	}
}

// topoSort runs Kahn's algorithm over "depends on" edges: buildOrder lists
// dependencies before dependents. Any node left unprocessed once no more
// zero-in-degree nodes remain is part of a cycle.
func topoSort(nodes []string, dependsOn map[string][]string) (order []string, cycleNodes []string, hasCycle bool) {
	inDegree := map[string]int{}
	dependents := map[string][]string{}
	for _, n := range nodes {
		inDegree[n] = 0
	}
	for n, deps := range dependsOn {
		inDegree[n] += len(deps)
		for _, d := range deps {
			dependents[d] = append(dependents[d], n)
		}
	}

	var queue []string
	for _, n := range nodes {
		if inDegree[n] == 0 {
			queue = append(queue, n)
		}
	}
	sort.Strings(queue)

	var result []string
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		result = append(result, n)
		var freed []string
		for _, dep := range dependents[n] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				freed = append(freed, dep)
			}
		}
		sort.Strings(freed)
		queue = append(queue, freed...)
	}

	if len(result) != len(nodes) {
		resulted := map[string]bool{}
		for _, n := range result {
			resulted[n] = true
		}
		for _, n := range nodes {
			if !resulted[n] {
				cycleNodes = append(cycleNodes, n)
			}
		}
		sort.Strings(cycleNodes)
		return nil, cycleNodes, true
	}
	return result, nil, false
}

// ---- validation ----

func validate(rawLines []string, p Parsed) []Issue {
	var issues []Issue

	nameCount := map[string]int{}
	nameIsDouble := map[string]bool{}
	for _, t := range p.Targets {
		nameCount[t.Name]++
		if t.IsDoubleColon {
			nameIsDouble[t.Name] = true
		}
	}
	for name, count := range nameCount {
		if count > 1 && !nameIsDouble[name] {
			issues = append(issues, Issue{
				Severity: "warning",
				Message:  fmt.Sprintf("target %q has %d single-colon rule definitions; only the last recipe wins in GNU Make", name, count),
			})
		}
	}

	for _, t := range p.Targets {
		for _, phony := range p.PhonyTargets {
			if phony == t.Name && len(t.Recipe) == 0 && len(t.Prerequisites) == 0 {
				_ = phony // declared phony with no body/deps is common and fine; no issue
			}
		}
	}
	for _, phony := range p.PhonyTargets {
		if nameCount[phony] == 0 {
			issues = append(issues, Issue{
				Severity: "warning",
				Message:  fmt.Sprintf(".PHONY declares %q but no rule defines it", phony),
			})
		}
	}

	// A line that looks like it was meant as a recipe (starts with spaces
	// immediately after a rule header, not a tab) is GNU Make's single
	// most common gotcha: it becomes a "missing separator" error at
	// runtime. We can't run make, but we can flag the shape.
	for i, line := range rawLines {
		if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "\t") {
			trimmed := strings.TrimLeft(line, " ")
			if trimmed != "" && i > 0 && !endsInContinuation(rawLines[i-1]) {
				prev := strings.TrimSpace(rawLines[i-1])
				if strings.HasSuffix(prev, ":") || isLikelyRuleHeader(prev) {
					issues = append(issues, Issue{
						Severity:   "warning",
						Message:    "line is indented with spaces directly under what looks like a rule header; GNU Make requires a literal tab to introduce a recipe line",
						LineNumber: i + 1,
					})
				}
			}
		}
	}

	return issues
}

// endsInContinuation reports whether line ends in an odd number of trailing
// backslashes, i.e. is itself continued onto the next physical line — used
// to avoid flagging a legitimate continuation line as a mis-indented recipe.
func endsInContinuation(line string) bool {
	n := 0
	for k := len(line) - 1; k >= 0 && line[k] == '\\'; k-- {
		n++
	}
	return n%2 == 1
}

func isLikelyRuleHeader(line string) bool {
	if line == "" || strings.HasPrefix(line, "#") {
		return false
	}
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return false
	}
	if idx+1 < len(line) && line[idx+1] == '=' {
		return false // variable assignment (:=), not a rule
	}
	return true
}

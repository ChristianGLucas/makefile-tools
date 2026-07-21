package nodes

import (
	"context"

	"christiangeorgelucas/makefile-tools/axiom"
	gen "christiangeorgelucas/makefile-tools/gen"
	"christiangeorgelucas/makefile-tools/internal/mkfile"
)

// GetDependencyGraph extracts the target-to-target dependency graph for
// build-order analysis: each target's prerequisites that are themselves
// targets in the file (plain source-file prerequisites are excluded — they
// have no build step to sequence), plus a topologically sorted build order.
// If the graph has a cycle, has_cycle is true, build_order is empty, and
// cycle_targets names the targets involved — never an infinite loop.
func GetDependencyGraph(ctx context.Context, ax axiom.Context, input *gen.MakefileInput) (*gen.DependencyGraph, error) {
	parsed, err := mkfile.Parse(input.GetContent())
	if err := checkBounds(err); err != nil {
		return nil, err
	}

	g := mkfile.BuildDependencyGraph(parsed.Targets)

	edges := make([]*gen.DependencyEdge, 0, len(g.Edges))
	for _, e := range g.Edges {
		edges = append(edges, &gen.DependencyEdge{Target: e.Target, DependsOnTargets: e.DependsOnTargets})
	}

	return &gen.DependencyGraph{
		Edges:        edges,
		BuildOrder:   g.BuildOrder,
		HasCycle:     g.HasCycle,
		CycleTargets: g.CycleTargets,
	}, nil
}

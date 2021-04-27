package lensvm

import (
	"github.com/lens-vm/gogl"
	"github.com/lens-vm/gogl/dfs"
	"github.com/lens-vm/gogl/graph/al"
)

type dependancyNode struct{}

type dependancyGraph struct {
	graph gogl.MutableLabeledGraph
}

func newDependancyGraph() *dependancyGraph {
	graph := gogl.Spec().
		// The graph should be mutable. Default is immutable.
		Mutable().
		// The graph should have directed edges (arcs). Default is undirected.
		Directed().
		// The graph's edges are plain - no labels, weights, etc. This is the default.
		Labeled().
		// No loops or parallel edges. This is the default.
		SimpleGraph().
		// al.G picks and returns an adjacency list-based graph, based on the spec.
		Create(al.G).
	// The builder always returns a Graph; type assert to get access to add/remove methods.
	(gogl.MutableLabeledGraph)

	return &dependancyGraph{graph}
}

// Both Module and Lens files define an import
// object, which imports named functions from
// lens modules.
//
// Eg. Lens A imports Lens B imports Lens C
// Then we have a depedancy graph like so:
// A ──> B ──> C
//
// However the graph can become more complicated
// multiple imports can be defined, and with
// different versions
//
//	   ┌─> B ──> D ───> H ─┐
// A ──┤				   ├──> I
//     └─> C ──> E ─┬─> F ─┘
//                  └─> G

func (vm *VM) makeDependancyGraph() error {
	vm.dgraph = newDependancyGraph()
	for id, mod := range vm.moduleImports {
		for lens, modDep := range mod.dependancies {
			vm.dgraph.graph.AddEdges(gogl.NewLabeledEdge(id, modDep.id, lens))
		}
	}
	return nil
}

func (d dependancyGraph) SortOrder(root string) ([]string, error) {
	var sortV []gogl.Vertex
	sortV, err := dfs.Toposort(d.graph, root)
	if err != nil {
		return nil, err
	}

	sortS := make([]string, len(sortV))
	for i, v := range sortV {
		sortS[i] = v.(string)
	}

	return sortS, nil
}

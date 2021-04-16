package lensvm

import (
	"github.com/lens-vm/gogl"
)

type dependancyNode struct{}

type dependancyGraph struct {
	graph gogl.Graph
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

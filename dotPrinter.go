package main

import "fmt"

// dotPrinter prints the dependency graph in dot format.
type dotPrinter struct{}

func (p dotPrinter) printPrologue() {
	// TODO: hardcoding the graph name will break when called with several files.
	fmt.Println("digraph G {")
}

func (p dotPrinter) printEpilogue() {
	fmt.Println("}")
}

func (p dotPrinter) printRootBin(bin string) {
	// nop
}

func (p dotPrinter) printDepBin(d *dependency) {
	// nop
}
func (p dotPrinter) printDep(from, to string) {
	fmt.Printf("\t\"%s\" -> \"%s\";\n", from, to)
}

package main

import "fmt"

// textPrinter prints dependencies like otool.
type textPrinter struct{ verbose bool }

func (p textPrinter) printPrologue() {
	// nop
}

func (p textPrinter) printEpilogue() {
	// nop
}

func (p textPrinter) printRootBin(bin string) {
	fmt.Printf("%s:\n", bin)
}

func (p textPrinter) printDepBin(d *dependency) {
	if p.verbose {
		fmt.Printf("\t%s %s\n", d.bin, d.info)
	} else {
		fmt.Printf("\t%s\n", d.bin)
	}
}
func (p textPrinter) printDep(from, to string) {
	// nop
}

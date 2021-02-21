// Totool (Transitive otool) is a thin wrapper over "otool -L" that displays both
// direct and transitive dependencies of a macOS mach-o binary.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	log.SetPrefix("totool: ")
	log.SetFlags(0)

	verbose := flag.Bool("v", false, "output extra info")
	dot := flag.Bool("dot", false, "generate dot output")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: totool [flags] file...\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	var pt printer
	if *dot {
		pt = dotPrinter{}
	} else {
		pt = textPrinter{*verbose}
	}

	for _, root := range args {
		err := walk(root, pt)
		if err != nil {
			log.Printf("%s: %v", root, err)
		}
	}
}

// dependency stores a single dependency found by otool.
type dependency struct {
	// path to binary
	bin string

	// additional data (versions...)
	info string
}

// A printer abstracts the rest of the program from the output layout.
type printer interface {
	// printPrologue is called before walking the dependency graph.
	printPrologue()

	// printRootBin is called to print the binary we want to print dependencies of.
	printRootBin(bin string)

	// printDepBin is called when walking into a new binary.
	printDepBin(d *dependency)

	// printDep is called to print a direct dependency between from and to binaries.
	printDep(from, to string)

	// printEpilogue is called after walking all nodes in the dependency graph.
	printEpilogue()
}

// walk traverses the graph of dependencies of the root binary in breadth-first
// order and call printer for each one.
func walk(root string, pt printer) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("cannot get %q absolute path: %v", root, err)
	}

	pt.printPrologue()
	defer pt.printEpilogue()

	toVisit := make([]dependency, 0)
	toVisit = append(toVisit, dependency{root, ""})

	visited := make(map[string]bool)

	for len(toVisit) > 0 {
		var from dependency
		from, toVisit = toVisit[0], toVisit[1:]
		if !visited[from.bin] {
			visited[from.bin] = true
			if from.bin == root {
				pt.printRootBin(root)
			} else {
				pt.printDepBin(&from)
			}
			i := len(toVisit)
			toVisit, err = appendDirectDeps(toVisit, from.bin)
			if err != nil {
				return err
			}
			for _, to := range toVisit[i:] {
				pt.printDep(from.bin, to.bin)
			}
		}
	}

	return nil
}

// depRe matches on otool output line.
// 	/usr/lib/libobjc.A.dylib (compatibility version 1.0.0, current version 228.0.0, upward)
var depRe = regexp.MustCompile(`\s*(.*)\s+(\(.*\))`)

// appendDirectDeps calls otool on bin and appends its dependencies to deps and
// returns the augmented slice.
func appendDirectDeps(deps []dependency, bin string) ([]dependency, error) {
	cmd := exec.Command("otool", "-L", bin)
	out, err := cmd.Output()
	if err != nil {
		err := err.(*exec.ExitError)
		fmt.Fprintf(os.Stderr, "%s", string(err.Stderr))
		return deps, fmt.Errorf("otool error when processing %s", bin)
	}

	s := bufio.NewScanner(bytes.NewReader(out))

	// Skip first line (the binary we are inspecting)
	s.Scan()

	for s.Scan() {
		sms := depRe.FindStringSubmatch(s.Text())
		if len(sms) != 3 {
			panic(fmt.Sprintf("unexpected otool output: %q, matched %v", s.Text(), sms))
		}
		depbin := resolveDepPath(bin, sms[1])
		if depbin != bin {
			deps = append(deps, dependency{depbin, sms[2]})
		} else {
			// The first dependency is the binary itself probably to display extra info about it.
			// Filter it out to avoid displaying self-edges in the graph.
		}
	}

	return deps, s.Err()
}

// resolveDepPath transforms a path emitted by otool representing a dependency
// of bin into an real path that can be fed back into otool.
func resolveDepPath(bin, path string) string {
	const relPrefix = "@executable_path/"
	if strings.HasPrefix(path, relPrefix) {
		bindir := filepath.Dir(bin) + string(filepath.Separator)
		return filepath.Clean(strings.Replace(path, relPrefix, bindir, 1))
	}
	return path
}

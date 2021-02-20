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

	for _, root := range args {
		err := walk(root, *verbose)
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

// walk traverses the graph of dependencies of the root binary in breadth-first
// order and prints them.
// If verbose is set, output extra per-dependency info.
func walk(root string, verbose bool) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("cannot get %q absolute path: %v", root, err)
	}

	toVisit := make([]dependency, 0)
	toVisit, err = appendDirectDeps(toVisit, root)
	if err != nil {
		return err
	}

	visited := make(map[string]bool)
	visited[root] = true

	fmt.Printf("%s:\n", root)
	for len(toVisit) > 0 {
		var d dependency
		d, toVisit = toVisit[0], toVisit[1:]
		if !visited[d.bin] {
			visited[d.bin] = true
			if verbose {
				fmt.Printf("\t%s %s\n", d.bin, d.info)
			} else {
				fmt.Printf("\t%s\n", d.bin)
			}
			toVisit, err = appendDirectDeps(toVisit, d.bin)
			if err != nil {
				return err
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
		deps = append(deps, dependency{resolveDepPath(bin, sms[1]), sms[2]})
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

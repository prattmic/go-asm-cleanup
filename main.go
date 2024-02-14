// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var dryRun = flag.Bool("dry-run", true, "set to false to actually edit files")

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(flag.Args()) != 1 {
		return fmt.Errorf("usage: fixasm GOROOT/src")
	}

	root := flag.Args()[0]
	fsys := os.DirFS(root)
	log.Printf("Processing %s...", root)

	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor trees.
		if d.Name() == "vendor" {
			return fs.SkipDir
		}

		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".s") {
			return nil
		}

		log.Printf("Assembly file %s", path)

		if err := processFile(root, path); err != nil {
			return fmt.Errorf("error processing %s: %w", path, err)
		}

		return nil
	})
}

func processFile(root, path string) error {
	components := strings.Split(path, "/")
	if len(components) < 2 {
		return fmt.Errorf("what")
	}
	components = components[:len(components)-1] // Remove filename
	pkg := strings.Join(components, "∕") // U+2215 DIVISION SLASH
	log.Printf("Package: %s", pkg)

	// Look for redundant package prefixes.
	re := regexp.MustCompile(regexp.QuoteMeta(pkg) + `·`)

	full := filepath.Join(root, path)

	if *dryRun {
		f, err := os.Open(full)
		if err != nil {
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if re.MatchString(line) {
				log.Printf("Found redundant package name in line: %s", line)
			}
		}
		return scanner.Err()
	}

	fi, err := os.Stat(full)
	if err != nil {
		return err
	}

	contents, err := os.ReadFile(full)
	if err != nil {
		return err
	}
	contents = re.ReplaceAll(contents, []byte(`·`))
	return os.WriteFile(full, contents, fi.Mode())
}

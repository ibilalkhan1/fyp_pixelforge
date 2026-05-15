package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ibilalkhan1/fyp_pixelforge/pixelforge_studio/capture"
)

const defaultRegressionDir = "tests/regressions"

// runTestSubcommand implements `pf-studio test [dir] [--regressions=path]`.
// Walks the regression directory two levels deep (project-hash → test-name)
// and invokes capture.ReplayRegression on each leaf.
func runTestSubcommand(args []string) {
	dir := defaultRegressionDir
	for _, a := range args {
		if strings.HasPrefix(a, "--regressions=") {
			dir = strings.TrimPrefix(a, "--regressions=")
			continue
		}
		if a != "" && !strings.HasPrefix(a, "-") {
			dir = a
		}
	}
	if _, err := os.Stat(dir); err != nil {
		log.Fatalf("pf-studio test: %s: %v", dir, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("pf-studio test: read %s: %v", dir, err)
	}
	failed := 0
	ran := 0
	for _, hashDir := range entries {
		if !hashDir.IsDir() {
			continue
		}
		hashPath := filepath.Join(dir, hashDir.Name())
		testDirs, err := os.ReadDir(hashPath)
		if err != nil {
			continue
		}
		for _, td := range testDirs {
			if !td.IsDir() {
				continue
			}
			testPath := filepath.Join(hashPath, td.Name())
			result, err := capture.ReplayRegression(testPath)
			ran++
			if err != nil {
				fmt.Printf("FAIL %s: %v\n", td.Name(), err)
				failed++
				continue
			}
			if !result.Passed {
				fmt.Printf("FAIL %s: %s\n", td.Name(), result.Detail)
				failed++
				continue
			}
			fmt.Printf("OK   %s\n", td.Name())
		}
	}
	fmt.Printf("ran %d regression(s); %d failed\n", ran, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

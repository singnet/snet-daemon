// coverage-filter removes generated sources and command entry points from a Go coverage profile.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var coverageRecord = regexp.MustCompile(`^(.+):[0-9]+\.[0-9]+,[0-9]+\.[0-9]+ [0-9]+ [0-9]+$`)

func filterProfile(root string, input io.Reader, output io.Writer) error {
	module, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	modulePath := ""
	for line := range strings.SplitSeq(string(module), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			modulePath = strings.Trim(fields[1], `"`)
			break
		}
	}
	if modulePath == "" {
		return fmt.Errorf("go.mod has no module directive")
	}
	scanner := bufio.NewScanner(input)
	if !scanner.Scan() {
		return fmt.Errorf("coverage profile is empty")
	}
	mode := strings.TrimPrefix(scanner.Text(), "\ufeff")
	if mode != "mode: atomic" && mode != "mode: count" && mode != "mode: set" {
		return fmt.Errorf("invalid coverage mode: %q", mode)
	}
	var result bytes.Buffer
	fmt.Fprintln(&result, mode)
	excluded := make(map[string]bool)
	records := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		match := coverageRecord.FindStringSubmatch(line)
		if match == nil {
			return fmt.Errorf("invalid coverage record: %q", line)
		}
		name := strings.ReplaceAll(match[1], `\`, "/")
		relative := strings.TrimPrefix(name, modulePath+"/")
		if !filepath.IsLocal(filepath.FromSlash(relative)) {
			return fmt.Errorf("source outside repository: %q", name)
		}
		if _, checked := excluded[relative]; !checked {
			skip := strings.HasSuffix(relative, ".pb.go") || relative == "snetd/main.go" ||
				strings.HasPrefix(relative, "resources/generate-smart-binds/") ||
				strings.HasPrefix(relative, "resources/coverage-filter/")
			if !skip {
				source, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, filepath.FromSlash(relative)), nil, parser.PackageClauseOnly|parser.ParseComments)
				if err != nil {
					return fmt.Errorf("inspect source %q: %w", relative, err)
				}
				skip = ast.IsGenerated(source)
			}
			excluded[relative] = skip
		}
		if excluded[relative] {
			continue
		}
		fmt.Fprintln(&result, line)
		records++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if records == 0 {
		return fmt.Errorf("coverage profile has no records after filtering")
	}
	_, err = result.WriteTo(output)
	return err
}

func writeProfile(root, inputPath, outputPath string) error {
	input, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.CreateTemp(filepath.Dir(outputPath), ".coverage-filtered-*.out")
	if err != nil {
		return err
	}
	defer os.Remove(output.Name())
	err = filterProfile(root, input, output)
	closeErr := output.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(output.Name(), outputPath)
}

func main() {
	root := flag.String("root", ".", "repository root containing go.mod")
	input := flag.String("input", "cover.out", "raw Go coverage profile")
	output := flag.String("output", "cover.filtered.out", "filtered Go coverage profile")
	flag.Parse()
	if err := writeProfile(*root, *input, *output); err != nil {
		fmt.Fprintln(os.Stderr, "Coverage filtering failed:", err)
		os.Exit(1)
	}
}

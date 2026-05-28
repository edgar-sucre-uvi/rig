package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/edgar-sucre-uvi/rig/pkg/forge"
	"golang.org/x/tools/imports"
)

func main() {
	typeName := flag.String("type", "", "The name of the interface to mock (Required)")
	outputFile := flag.String("outfile", "", "The output file name (Optional)")
	outDir := flag.String("outdir", "", "The destination folder path (Optional)")
	flag.Parse()

	if *typeName == "" {
		fmt.Fprintln(os.Stderr, "Error: -type is required.")
		os.Exit(1)
	}

	goFile := os.Getenv("GOFILE")
	if goFile == "" {
		fmt.Fprintln(os.Stderr, "Error: rig-mock must be run via 'go:generate'")
		os.Exit(1)
	}

	interfaceDef, err := forge.ParseInterface(goFile, *typeName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Rig parsing error: %v\n", err)
		os.Exit(1)
	}

	finalOutDir := *outDir
	if finalOutDir == "" {
		finalOutDir = filepath.Join(filepath.Dir(goFile), "mocks")
	}

	absOutDir, _ := filepath.Abs(finalOutDir)
	packageName := filepath.Base(absOutDir)

	// NEW: If output folder is external, rewrite types from *User to *domain.User
	if absOutDir != interfaceDef.SourcePath {
		interfaceDef.QualifyTypes()
	}

	templateCtx := struct {
		Package string
		forge.InterfaceDef
	}{
		Package:      packageName,
		InterfaceDef: *interfaceDef,
	}

	var buf bytes.Buffer

	if err := mockTemplate.Execute(&buf, templateCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Rig template error: %v\n", err)
		os.Exit(1)
	}

	// Dynamic, intelligent imports processing step
	outName := *outputFile
	if outName == "" {
		outName = fmt.Sprintf("%s_mock_gen.go", strings.ToLower(*typeName))
	}
	finalPath := filepath.Join(finalOutDir, outName)

	// imports.Process automatically removes unused imports and resolves standard library ones!
	formattedCode, err := imports.Process(finalPath, buf.Bytes(), &imports.Options{
		Comments:   true,
		TabIndent:  true,
		TabWidth:   8,
		FormatOnly: false, // Set to false so it completely calculates missing imports
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Rig imports/formatting error: %v\n", err)
		_ = os.WriteFile("debug_failed_output.go", buf.Bytes(), 0644)
		os.Exit(1)
	}

	if err := os.MkdirAll(finalOutDir, 0755); err != nil {
		os.Exit(1)
	}

	if err := os.WriteFile(finalPath, formattedCode, 0644); err != nil {
		os.Exit(1)
	}

	fmt.Printf("Rig: Forged mock with resolved imports at %s\n", finalPath)
}

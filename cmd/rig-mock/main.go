package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/edgar-sucre-uvi/rig/pkg/forge"
)

func main() {
	// 1. Define standard, industrial CLI flags
	typeName := flag.String("type", "", "The name of the interface to mock (Required)")
	outputFile := flag.String("output", "", "The output file name (Optional, defaults to <type>_mock_gen.go)")
	outDir := flag.String("outdir", "", "The destination folder path for the mock (Optional, defaults to ./mocks)")
	flag.Parse()

	if *typeName == "" {
		fmt.Fprintln(os.Stderr, "Error: the -type flag is required.")
		flag.Usage()
		os.Exit(1)
	}

	// 2. Grab the environment variables injected by 'go:generate'
	goFile := os.Getenv("GOFILE")
	if goFile == "" {
		fmt.Fprintln(os.Stderr, "Error: rig-mock must be run via 'go:generate'")
		os.Exit(1)
	}

	// 3. Fire up the shared AST engine to inspect the interface
	fmt.Printf("Rig: Mining file %s for interface %s...\n", goFile, *typeName)
	interfaceDef, err := forge.ParseInterface(goFile, *typeName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Rig parsing error: %v\n", err)
		os.Exit(1)
	}

	// 4. Calculate the output path and extract the target package name
	finalOutDir := *outDir
	if finalOutDir == "" {
		finalOutDir = filepath.Join(filepath.Dir(goFile), "mocks")
	}

	// Clean up path details to accurately determine the directory name
	absOutDir, err := filepath.Abs(finalOutDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Rig path resolution error: %v\n", err)
		os.Exit(1)
	}
	packageName := filepath.Base(absOutDir)

	// Inject the dynamic package name into our template data context
	// (We pass an anonymous struct so the template can read .Package and .Interface)
	templateCtx := struct {
		Package string
		forge.InterfaceDef
	}{
		Package:      packageName,
		InterfaceDef: *interfaceDef,
	}

	// 5. Run the data through our text/template engine
	var buf bytes.Buffer
	if err := mockTemplate.Execute(&buf, templateCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Rig template error: %v\n", err)
		os.Exit(1)
	}

	// 6. Clean up the output using Go's official source code formatter
	formattedCode, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Rig code-formatting error: %v\n", err)
		_ = os.WriteFile("debug_failed_output.go", buf.Bytes(), 0644)
		os.Exit(1)
	}

	// 7. Calculate file name and drop it to disk
	outName := *outputFile
	if outName == "" {
		outName = fmt.Sprintf("%s_mock_gen.go", strings.ToLower(*typeName))
	}

	if err := os.MkdirAll(finalOutDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Rig failed to create directory %s: %v\n", finalOutDir, err)
		os.Exit(1)
	}

	finalPath := filepath.Join(finalOutDir, outName)
	if err := os.WriteFile(finalPath, formattedCode, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Rig failed to write generated file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Rig: Successfully forged mock in package '%s' at %s\n", packageName, finalPath)
}

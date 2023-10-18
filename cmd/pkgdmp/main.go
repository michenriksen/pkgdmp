package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/michenriksen/pkgdmp"
	"github.com/michenriksen/pkgdmp/internal/cli"

	"github.com/alecthomas/chroma/quick"
)

func main() {
	cfg, exitCode, err := cli.ParseFlags(os.Args[1:], os.Stderr)
	if err != nil {
		os.Exit(exitCode)
	}

	pkgParserOpts, err := cli.ParserOptsFromCfg(cfg)
	if err != nil {
		log.Fatal(err)
	}

	pkgParser := pkgdmp.NewParser(pkgParserOpts)

	unparsed, err := getPackages(cfg.Dirs)
	if err != nil {
		log.Fatal(err)
	}

	parsed := make([]*pkgdmp.Package, 0, len(unparsed))

	for _, uPkg := range unparsed {
		pkg, err := pkgParser.Package(doc.New(uPkg, "", doc.AllDecls))
		if err != nil {
			log.Fatal(err)
		}

		parsed = append(parsed, pkg)
	}

	if err := printPackages(parsed, cfg); err != nil {
		log.Fatal(err)
	}
}

func getPackages(dirs []string) ([]*ast.Package, error) {
	var all []*ast.Package

	for _, dir := range dirs {
		fset := token.NewFileSet()

		pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parsing files in %s: %w", dir, err)
		}

		for _, pkg := range pkgs {
			all = append(all, pkg)
		}
	}

	return all, nil
}

func printPackages(pkgs []*pkgdmp.Package, cfg *cli.Config) error {
	if cfg.JSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		if err := encoder.Encode(pkgs); err != nil {
			return fmt.Errorf("encoding packages as JSON: %w", err)
		}

		return nil
	}

	for _, pkg := range pkgs {
		source, err := pkg.Source()
		if err != nil {
			return fmt.Errorf("getting source for %s package: %w", pkg.Name, err)
		}

		if cfg.NoHighlight {
			fmt.Printf("%s\n\n", source)
			continue
		}

		highlighted, err := highlight(source, cfg.Theme)
		if err != nil {
			return fmt.Errorf("syntax highlighting source for %s package: %w", pkg.Name, err)
		}

		fmt.Printf("%s\n\n", highlighted)
	}

	return nil
}

func highlight(source, theme string) (string, error) {
	var b strings.Builder

	if err := quick.Highlight(&b, source, "go", "terminal", theme); err != nil {
		return "", fmt.Errorf("chroma error: %w", err)
	}

	return b.String(), nil
}

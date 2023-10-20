package pkgdmp_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"go/doc"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/michenriksen/pkgdmp"
)

const defaultPkgName = "mypackage"

var updateGolden = flag.Bool("update-golden", false, "update golden test files")

var defaultDocPkg *doc.Package

type parserTestCase struct {
	name       string
	sourceFile string // File in testdata/source or empty for default.
	goldenFile string // File in testdata/golden or empty for default.
	opts       []pkgdmp.ParserOption
}

func TestMain(m *testing.M) {
	flag.Parse()

	initDefaultDocPkg()

	os.Exit(m.Run())
}

func TestParser_GoldenFiles_Unique(t *testing.T) {
	if *updateGolden {
		t.Skip("golden files are being updated")
	}

	files, err := filepath.Glob(filepath.Join("testdata", "golden", "parser_*.golden"))
	if err != nil {
		t.Fatalf("error getting golden files list: %v", err)
	}

	if len(files) == 0 {
		t.Skip("no golden files found")
	}

	hMap := make(map[string]string)

	for _, file := range files {
		basefile := filepath.Base(file)

		t.Run(basefile+" is unique", func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("error reading golden file: %v", err)
			}

			if len(data) == 0 {
				t.Errorf("expected golden files to have content, but %s is empty", basefile)
			}

			h := sha256.New()
			h.Write(data)

			chksm := base64.StdEncoding.EncodeToString(h.Sum(nil))

			if otherFile, found := hMap[chksm]; found {
				t.Errorf("expected golden files to be unique, but %s and %s are identical (SHA256: %s)",
					basefile,
					otherFile,
					chksm,
				)
			}

			hMap[chksm] = basefile
		})
	}
}

func TestParser_Package(t *testing.T) {
	tt := []*parserTestCase{
		{
			name: "default options",
			opts: nil,
		},
		{
			name: "full doc comments",
			opts: []pkgdmp.ParserOption{pkgdmp.WithFullDocs()},
		},
		{
			name: "exclude doc comments",
			opts: []pkgdmp.ParserOption{pkgdmp.WithNoDocs()},
		},
		{
			name: "exclude unexported",
			opts: []pkgdmp.ParserOption{
				pkgdmp.WithSymbolFilters(
					pkgdmp.FilterUnexported(pkgdmp.Exclude),
				),
			},
		},
		{
			name: "exclude structs",
			opts: []pkgdmp.ParserOption{
				pkgdmp.WithSymbolFilters(
					pkgdmp.FilterSymbolTypes(pkgdmp.Exclude, pkgdmp.SymbolStructType),
				),
			},
		},
		{
			name: "exclude funcs",
			opts: []pkgdmp.ParserOption{
				pkgdmp.WithSymbolFilters(
					pkgdmp.FilterSymbolTypes(pkgdmp.Exclude, pkgdmp.SymbolFunc),
				),
			},
		},
		{
			name: "exclude func types",
			opts: []pkgdmp.ParserOption{
				pkgdmp.WithSymbolFilters(
					pkgdmp.FilterSymbolTypes(pkgdmp.Exclude, pkgdmp.SymbolFuncType),
				),
			},
		},
		{
			name: "exclude interfaces",
			opts: []pkgdmp.ParserOption{
				pkgdmp.WithSymbolFilters(
					pkgdmp.FilterSymbolTypes(pkgdmp.Exclude, pkgdmp.SymbolInterfaceType),
				),
			},
		},
		{
			name: "matching idents",
			opts: []pkgdmp.ParserOption{
				pkgdmp.WithSymbolFilters(
					pkgdmp.FilterMatchingIdents(pkgdmp.Include, regexp.MustCompile(`^My(Other)?Function$`)),
				),
			},
		},
		{
			name: "exclude matching idents",
			opts: []pkgdmp.ParserOption{
				pkgdmp.WithSymbolFilters(
					pkgdmp.FilterMatchingIdents(pkgdmp.Exclude, regexp.MustCompile(`my\w+Interface`)),
				),
			},
		},
		{
			name: "match and exclude match pattern",
			opts: []pkgdmp.ParserOption{
				pkgdmp.WithSymbolFilters(
					pkgdmp.FilterMatchingIdents(pkgdmp.Include, regexp.MustCompile(`^My\w*Function$`)),
					pkgdmp.FilterMatchingIdents(pkgdmp.Exclude, regexp.MustCompile(`^MyOtherFunction$`)),
				),
			},
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			tc.run(t)
		})
	}
}

func (tc *parserTestCase) run(tb *testing.T) {
	tb.Helper()

	pkgParser, _ := pkgdmp.NewParser(tc.opts...)

	pkg, err := pkgParser.Package(tc.pkgDoc(tb))
	if err != nil {
		tb.Errorf("expected no error when parsing package, but got: %v", err)
	}

	tc.compareGolden(tb, pkg)
}

func (tc *parserTestCase) pkgDoc(tb testing.TB) *doc.Package {
	tb.Helper()

	if tc.sourceFile == "" {
		return defaultDocPkg
	}

	tDir := tb.TempDir()

	src, err := os.Open(filepath.Join("testdata", tc.sourceFile))
	if err != nil {
		tb.Fatalf("error opening source file: %v", err)
	}
	defer src.Close()

	dst, err := os.Create(filepath.Join(tDir, "file.go"))
	if err != nil {
		tb.Fatalf("error creating temporary source file: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		tb.Fatalf("error copying source file content to temporary file")
	}

	fset := token.NewFileSet()

	pkgMap, err := parser.ParseDir(fset, tDir, nil, parser.ParseComments)
	if err != nil {
		tb.Fatalf("error parsing source: %v", err)
	}

	pkg, ok := pkgMap[defaultPkgName]
	if !ok {
		tb.Fatalf("expected source to specify package %q", defaultPkgName)
	}

	return doc.New(pkg, "", doc.AllDecls)
}

func (tc *parserTestCase) compareGolden(tb testing.TB, pkg *pkgdmp.Package) {
	tb.Helper()

	actual, err := pkg.Source()
	if err != nil {
		tb.Errorf("expected no error when getting parsed package source, but got: %v", err)
	}

	goldenFile := tc.goldenFilepath()

	if *updateGolden {
		if err := os.WriteFile(goldenFile, []byte(actual), 0o600); err != nil {
			tb.Fatalf("error updating golden file: %v", err)
		}

		tb.Logf("updated golden file %s", goldenFile)
	}

	data, err := os.ReadFile(goldenFile)
	if err != nil {
		tb.Fatalf("error reading golden file: %v", err)
	}

	golden := string(data)

	if actual != golden {
		diff, err := tc.diffGolden(tb, actual)
		if err != nil {
			tb.Logf("ERROR: diff failed with error %q; using fallback comparison", err)
			diff = fmt.Sprintf("GOLDEN:\n\n%s\n\nACTUAL:\n\n%s\n", golden, actual)
		}

		tb.Errorf(
			"expected package source to be golden (run with '-update-golden' flag to update)\n\n%s\n", diff)
	}
}

func (tc *parserTestCase) diffGolden(tb testing.TB, actual string) (string, error) {
	tb.Helper()

	actualPath := filepath.Join(tb.TempDir(), "actual.go")

	if err := os.WriteFile(actualPath, []byte(actual), 0o600); err != nil {
		return "", fmt.Errorf("writing actual source to temporary file: %w", err)
	}

	goldenPath, err := filepath.Abs(tc.goldenFilepath())
	if err != nil {
		return "", fmt.Errorf("converting golden file path to absolute path: %w", err)
	}

	cmd := exec.Command(
		"git", "--no-pager",
		"diff", "--no-index", "--exit-code", "--color=always", goldenPath, actualPath,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr := &exec.ExitError{}
		if !errors.As(err, &exitErr) {
			return "", fmt.Errorf("running git diff command: %w\n\nOutput:\n\n%s", err, string(out))
		}
	}

	diff := bytes.Join(bytes.Split(out, []byte("\n"))[5:], []byte("\n"))

	return fmt.Sprintf("DIFF:\n\n%s\n", string(diff)), nil
}

func (tc *parserTestCase) goldenFilepath() string {
	name := tc.goldenFile
	if name == "" {
		name = tc.name
	}

	name = "parser_" + strings.ReplaceAll(name, " ", "-") + ".golden"

	return filepath.Join("testdata", "golden", strings.ToLower(name))
}

func initDefaultDocPkg() {
	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(
		fset,
		filepath.Join("testdata", "source"),
		func(fi fs.FileInfo) bool { return filepath.Base(fi.Name()) == "default.go" },
		parser.ParseComments,
	)
	if err != nil {
		panic(fmt.Errorf("error parsing default source file: %w", err))
	}

	pkg, ok := pkgs[defaultPkgName]
	if !ok {
		panic(fmt.Errorf("default source file does not specify expected %q package", defaultPkgName))
	}

	defaultDocPkg = doc.New(pkg, "", doc.AllDecls)
}

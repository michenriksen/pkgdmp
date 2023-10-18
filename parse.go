package pkgdmp

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/token"
	"regexp"
	"strings"
)

// ParserOptions represents the options for [Parser].
type ParserOptions struct {
	ExcludeRegexp     *regexp.Regexp // Don't parse entities with names matching regexp.
	OnlyRegexp        *regexp.Regexp // Only parse entities with names matching regexp.
	ExcludeDocs       bool           // Don't parse entity doc comments.
	ExcludeFuncTypes  bool           // Don't parse function types.
	ExcludeFuncs      bool           // Don't parse functions.
	ExcludeInterfaces bool           // Don't parse interfaces.
	ExcludeStructs    bool           // Don't parse structs.
	FullDocs          bool           // Include full doc comments instead of synopsis.
	Unexported        bool           // Parse unexported entities.
}

// Parser parses go packages to simple structs.
type Parser struct {
	opts ParserOptions
}

// NewParser returns a parser configured with opts.
func NewParser(opts ParserOptions) *Parser {
	return &Parser{opts: opts}
}

// Package parses dPkg to a simplified [Package].
func (p *Parser) Package(dPkg *doc.Package) (Package, error) {
	pkg := Package{
		Name: dPkg.Name,
		Doc:  p.mkDoc(dPkg.Doc),
	}

	if err := p.parseFuncs(&pkg, dPkg.Funcs); err != nil {
		return Package{}, fmt.Errorf("parsing functions: %w", err)
	}

	if err := p.parseTypes(&pkg, dPkg.Types); err != nil {
		return Package{}, fmt.Errorf("parsing types: %w", err)
	}

	return pkg, nil
}

func (p *Parser) parseFuncs(pkg *Package, fns []*doc.Func) error {
	if p.opts.ExcludeFuncs {
		return nil
	}

	for _, fn := range fns {
		if !p.includeIdent(fn.Name) {
			continue
		}

		pkg.Funcs = append(pkg.Funcs, p.parseFunc(fn))
	}

	return nil
}

func (p *Parser) parseTypes(pkg *Package, types []*doc.Type) error {
	for _, t := range types {
		if t.Decl.Tok != token.TYPE {
			continue
		}

		for _, spec := range t.Decl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			if !p.includeType(typeSpec) {
				continue
			}

			td := TypeDef{
				Name: t.Name,
				Doc:  p.mkDoc(t.Doc),
			}

			if err := p.parseFuncs(pkg, t.Funcs); err != nil {
				return fmt.Errorf("parsing functions for %s type: %w", t.Name, err)
			}

			for _, m := range t.Methods {
				if !p.includeMethod(m.Name) {
					continue
				}

				td.Methods = append(td.Methods, p.parseFunc(m))
			}

			switch ts := typeSpec.Type.(type) {
			case *ast.Ident:
				td.Type = ts.Name
			case *ast.StructType:
				td.Type = "struct"
				td.Fields = p.parseStructFieldList(ts.Fields)
			case *ast.InterfaceType:
				td.Type = "interface"

				if ts.Methods != nil {
					for _, m := range ts.Methods.List {
						ft, ok := m.Type.(*ast.FuncType)
						if !ok {
							continue
						}

						f := Func{
							Name:    m.Names[0].Name,
							Params:  p.parseFieldList(ft.Params),
							Results: p.parseFieldList(ft.Results),
							funcKw:  false,
						}

						if m.Doc != nil {
							f.Doc = p.mkDoc(m.Doc.Text())
						}

						if m.Comment != nil {
							f.Comment = p.mkDoc(m.Comment.Text())
						}

						td.Methods = append(td.Methods, f)
					}
				}
			case *ast.FuncType:
				td.Type = "func"
				td.Params = p.parseFieldList(ts.Params)
				td.Results = p.parseFieldList(ts.Results)
			case *ast.MapType:
				td.Type = "map"
				td.Key = printNodes(ts.Key)
				td.Value = printNodes(ts.Value)
			case *ast.ChanType:
				td.Type = "chan"
				td.Value = printNodes(ts.Value)

				switch ts.Dir {
				case ast.RECV:
					td.Dir = "recv"
				case ast.SEND:
					td.Dir = "send"
				}
			case *ast.ArrayType:
				td.Type = "array"
				td.Elt = printNodes(ts.Elt)

				if ts.Len != nil {
					td.Len = printNodes(ts.Len)
				}
			default:
				continue
			}

			pkg.Types = append(pkg.Types, td)
		}
	}

	return nil
}

func (p *Parser) parseFunc(df *doc.Func) Func {
	decl := df.Decl

	fn := Func{
		Name:   df.Name,
		Doc:    p.mkDoc(df.Doc),
		funcKw: decl.Type.Func != token.NoPos,
	}

	if decl.Recv != nil && decl.Recv.NumFields() != 0 {
		fr := p.parseField(decl.Recv.List[0])
		fn.Receiver = &fr
	}

	if decl.Type.Params != nil && decl.Type.Params.NumFields() != 0 {
		fn.Params = p.parseFieldList(decl.Type.Params)
	}

	if decl.Type.Results != nil && decl.Type.Results.NumFields() != 0 {
		fn.Results = p.parseFieldList(decl.Type.Results)
	}

	return fn
}

func (p *Parser) parseFieldList(fl *ast.FieldList) []Field {
	if fl == nil {
		return nil
	}

	res := make([]Field, len(fl.List))

	for i, f := range fl.List {
		res[i] = p.parseField(f)
	}

	return res
}

func (p *Parser) parseStructFieldList(fl *ast.FieldList) []Field {
	if fl == nil {
		return nil
	}

	res := make([]Field, 0, len(fl.List))

	for _, f := range fl.List {
		if !p.includeStructField(f.Names[0].Name) {
			continue
		}

		res = append(res, p.parseField(f))
	}

	return res
}

func (p *Parser) parseField(af *ast.Field) Field {
	f := Field{
		Names: identNames(af.Names),
		Type:  printNodes(af.Type),
	}

	if af.Doc != nil {
		f.Doc = p.mkDoc(af.Doc.Text())
	}

	if af.Comment != nil {
		f.Comment = p.mkDoc(af.Comment.Text())
	}

	return f
}

func (p *Parser) includeIdent(name string) bool {
	if !isExportedIdent(name) && !p.opts.Unexported {
		return false
	}

	return (p.opts.OnlyRegexp == nil || p.opts.OnlyRegexp.MatchString(name)) &&
		(p.opts.ExcludeRegexp == nil || !p.opts.ExcludeRegexp.MatchString(name))
}

func (p *Parser) includeType(at *ast.TypeSpec) bool {
	if !p.includeIdent(at.Name.Name) {
		return false
	}

	switch at.Type.(type) {
	case *ast.StructType:
		return !p.opts.ExcludeStructs
	case *ast.FuncType:
		return !p.opts.ExcludeFuncTypes
	case *ast.InterfaceType:
		return !p.opts.ExcludeInterfaces
	default:
		return true
	}
}

func (p *Parser) includeMethod(name string) bool {
	return isExportedIdent(name) || p.opts.Unexported
}

func (p *Parser) includeStructField(name string) bool {
	return isExportedIdent(name) || p.opts.Unexported
}

func (p *Parser) mkDoc(fullDoc string) string {
	if p.opts.ExcludeDocs {
		return ""
	}

	fullDoc = strings.TrimPrefix(strings.TrimSpace(fullDoc), "// ")

	if p.opts.FullDocs {
		return fullDoc
	}

	pkg := doc.Package{}

	return pkg.Synopsis(fullDoc)
}

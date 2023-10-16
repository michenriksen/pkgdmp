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

			switch ts := typeSpec.Type.(type) {
			case *ast.StructType:
				if !p.opts.ExcludeStructs && p.includeIdent(t.Name) {
					s := p.parseStruct(t.Name, ts, t.Doc)

					for _, m := range t.Methods {
						if !p.includeMethod(m.Name) {
							continue
						}

						s.Funcs = append(s.Funcs, p.parseFunc(m))
					}

					pkg.Structs = append(pkg.Structs, s)
				}

				if err := p.parseFuncs(pkg, t.Funcs); err != nil {
					return fmt.Errorf("parsing functions for %s struct: %w", t.Name, err)
				}
			case *ast.InterfaceType:
				if !p.opts.ExcludeInterfaces && p.includeIdent(t.Name) {
					pkg.Interfaces = append(pkg.Interfaces, p.parseInterface(t.Name, ts, t.Doc))
				}

				if err := p.parseFuncs(pkg, t.Funcs); err != nil {
					return fmt.Errorf("parsing functions for %s interface: %w", t.Name, err)
				}
			case *ast.FuncType:
				if !p.opts.ExcludeFuncTypes && p.includeIdent(t.Name) {
					pkg.FuncTypes = append(pkg.FuncTypes, p.parseFuncType(t.Name, ts, t.Doc))
				}

				if err := p.parseFuncs(pkg, t.Funcs); err != nil {
					return fmt.Errorf("parsing %s function type implementations: %w", t.Name, err)
				}
			}
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
		fr := p.parseFuncReceiver(decl.Recv.List[0])
		fn.Receiver = &fr
	}

	if decl.Type.Params != nil && decl.Type.Params.NumFields() != 0 {
		fn.Params = make([]FuncParam, decl.Type.Params.NumFields())

		for i, param := range decl.Type.Params.List {
			fn.Params[i] = p.parseFuncParam(param)
		}
	}

	if decl.Type.Results != nil && decl.Type.Results.NumFields() != 0 {
		fn.Results = make([]FuncResult, decl.Type.Results.NumFields())

		for i, r := range decl.Type.Results.List {
			fn.Results[i] = p.parseFuncResult(r)
		}
	}

	return fn
}

func (p *Parser) parseFuncReceiver(f *ast.Field) FuncReceiver {
	fr := FuncReceiver{Type: p.parseType(f.Type)}

	if len(f.Names) != 0 {
		fr.Name = f.Names[0].Name
	}

	return fr
}

func (p *Parser) parseFuncParam(f *ast.Field) FuncParam {
	return FuncParam{
		Names: identNames(f.Names),
		Type:  p.parseType(f.Type),
	}
}

func (p *Parser) parseFuncResult(f *ast.Field) FuncResult {
	return FuncResult{
		Names: identNames(f.Names),
		Type:  p.parseType(f.Type),
	}
}

func (p *Parser) parseStruct(name string, as *ast.StructType, fullDoc string) StructType {
	s := StructType{
		Name: name,
		Doc:  p.mkDoc(fullDoc),
	}

	if as.Fields == nil || as.Fields.NumFields() == 0 {
		return s
	}

	s.Fields = make([]StructField, 0, as.Fields.NumFields())

	for _, f := range as.Fields.List {
		if !p.opts.Unexported && !isExportedIdent(f.Names[0].Name) {
			continue
		}

		sf := StructField{
			Names: identNames(f.Names),
			Type:  p.parseType(f.Type),
			Doc:   p.mkDoc(f.Doc.Text()),
		}

		if f.Comment != nil && len(f.Comment.List) != 0 {
			sf.Comment = p.mkDoc(f.Comment.List[0].Text)
		}

		s.Fields = append(s.Fields, sf)
	}

	return s
}

func (p *Parser) parseFuncType(name string, aft *ast.FuncType, fullDoc string) FuncType {
	ft := FuncType{
		Name: name,
		Doc:  p.mkDoc(fullDoc),
	}

	if aft.Params != nil && aft.Params.NumFields() != 0 {
		ft.Params = make([]FuncParam, aft.Params.NumFields())

		for i, param := range aft.Params.List {
			ft.Params[i] = p.parseFuncParam(param)
		}
	}

	if aft.Results != nil && aft.Results.NumFields() != 0 {
		ft.Results = make([]FuncResult, aft.Results.NumFields())

		for i, res := range aft.Results.List {
			ft.Results[i] = p.parseFuncResult(res)
		}
	}

	return ft
}

func (p *Parser) parseInterface(name string, aiface *ast.InterfaceType, fullDoc string) InterfaceType {
	iface := InterfaceType{
		Name: name,
		Doc:  p.mkDoc(fullDoc),
	}

	if aiface.Methods == nil || aiface.Methods.NumFields() == 0 {
		return iface
	}

	iface.Funcs = make([]Func, aiface.Methods.NumFields())

	for i, m := range aiface.Methods.List {
		fn := Func{Name: m.Names[0].Name}

		ft, ok := m.Type.(*ast.FuncType)
		if !ok {
			panic(fmt.Errorf("failed asserting %s interface method %s as *ast.FuncType", name, fn.Name))
		}

		if ft.Params != nil && ft.Params.NumFields() != 0 {
			fn.Params = make([]FuncParam, ft.Params.NumFields())

			for j, param := range ft.Params.List {
				fn.Params[j] = p.parseFuncParam(param)
			}
		}

		if ft.Results != nil && ft.Results.NumFields() != 0 {
			fn.Results = make([]FuncResult, ft.Results.NumFields())

			for j, res := range ft.Results.List {
				fn.Results[j] = p.parseFuncResult(res)
			}
		}

		iface.Funcs[i] = fn
	}

	return iface
}

func (p *Parser) parseType(node ast.Node) Type {
	switch t := node.(type) {
	case *ast.Ident:
		return Type{Name: t.Name}
	case *ast.StarExpr:
		return Type{Prefix: "*", Name: p.parseType(t.X).String()}
	case *ast.Ellipsis:
		return Type{Prefix: "...", Name: p.parseType(t.Elt).String()}
	case *ast.SelectorExpr:
		return Type{Prefix: p.parseType(t.X).String() + ".", Name: t.Sel.Name}
	case *ast.ArrayType:
		return Type{Prefix: "[]", Name: p.parseType(t.Elt).String()}
	case *ast.MapType:
		return Type{Name: "map[" + p.parseType(t.Key).String() + "]" + p.parseType(t.Value).String()}
	case *ast.FuncType:
		return Type{Name: p.funcTypeAsString(t)}
	case *ast.ChanType:
		ret := Type{Name: p.parseType(t.Value).String()}

		switch t.Dir {
		case ast.RECV:
			ret.Prefix = "<-chan "
		case ast.SEND:
			ret.Prefix = "chan<- "
		default:
			ret.Prefix = "chan "
		}

		return ret
	default:
		return Type{Name: fmt.Sprintf("??%T", t)}
	}
}

func (p *Parser) funcTypeAsString(fn *ast.FuncType) string {
	var b strings.Builder

	if fn.Func != token.NoPos {
		b.WriteString("func")
	}

	if fn.Params != nil {
		params := make([]string, len(fn.Params.List))

		for i, param := range fn.Params.List {
			params[i] = p.parseFuncParam(param).String()
		}

		fmt.Fprintf(&b, "(%s)", strings.Join(params, ", "))
	}

	if fn.Results != nil && len(fn.Results.List) != 0 {
		results := make([]string, len(fn.Results.List))

		for i, r := range fn.Results.List {
			results[i] = p.parseFuncResult(r).String()
		}

		if len(results) == 1 {
			b.WriteString(" " + results[0])

			return b.String()
		}

		fmt.Fprintf(&b, " (%s)", strings.Join(results, ", "))
	}

	return b.String()
}

func (p *Parser) includeIdent(name string) bool {
	if !isExportedIdent(name) && !p.opts.Unexported {
		return false
	}

	return (p.opts.OnlyRegexp == nil || p.opts.OnlyRegexp.MatchString(name)) &&
		(p.opts.ExcludeRegexp == nil || !p.opts.ExcludeRegexp.MatchString(name))
}

func (p *Parser) includeMethod(name string) bool {
	return isExportedIdent(name) || p.opts.Unexported
}

func (p *Parser) mkDoc(fullDoc string) string {
	if p.opts.ExcludeDocs {
		return ""
	}

	fullDoc = strings.TrimPrefix(fullDoc, "// ")

	if p.opts.FullDocs {
		return fullDoc
	}

	pkg := doc.Package{}

	return pkg.Synopsis(fullDoc)
}

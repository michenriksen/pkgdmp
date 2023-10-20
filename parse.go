package pkgdmp

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/token"
	"hash/fnv"
	"strings"
)

var typeNames = map[token.Token]string{
	token.INT:    "int",
	token.FLOAT:  "float64",
	token.IMAG:   "complex128",
	token.CHAR:   "rune",
	token.STRING: "string",
}

// ParserOption configures a [Parser].
type ParserOption interface {
	Fingerprint() uint64
	apply(*Parser) error
}

// Parser parses go packages to simple structs.
type Parser struct {
	filters  []SymbolFilter
	fullDocs bool
	noDocs   bool
}

// NewParser returns a parser configured with options.
func NewParser(opts ...ParserOption) (*Parser, error) {
	p := &Parser{}

	for _, opt := range opts {
		if err := opt.apply(p); err != nil {
			return nil, fmt.Errorf("applying parser option: %w", err)
		}
	}

	return p, nil
}

// Package parses dPkg to a simplified [Package].
func (p *Parser) Package(dPkg *doc.Package) (*Package, error) {
	pkg := &Package{
		Name: dPkg.Name,
		Doc:  p.mkDoc(dPkg.Doc),
	}

	if err := p.parseConsts(pkg, dPkg.Consts); err != nil {
		return nil, fmt.Errorf("parsing constants: %w", err)
	}

	if err := p.parseTypes(pkg, dPkg.Types); err != nil {
		return nil, fmt.Errorf("parsing types: %w", err)
	}

	if err := p.parseFuncs(pkg, dPkg.Funcs); err != nil {
		return nil, fmt.Errorf("parsing functions: %w", err)
	}

	return pkg, nil
}

func (p *Parser) parseConsts(pkg *Package, cnsts []*doc.Value) error {
	for _, dVal := range cnsts {
		cg := p.parseConst(dVal)
		if len(cg.Consts) == 0 {
			continue
		}

		pkg.Consts = append(pkg.Consts, cg)
	}

	return nil
}

func (p *Parser) parseConst(dVal *doc.Value) ConstGroup {
	cg := ConstGroup{Doc: p.mkDoc(dVal.Doc)}

	for _, s := range dVal.Decl.Specs {
		vs, ok := s.(*ast.ValueSpec)
		if !ok {
			panic(fmt.Errorf("unsupported const spec type %T", s))
		}

		c := Const{
			Names:   identNames(vs.Names),
			Values:  make([]Value, 0, len(vs.Values)),
			valSpec: vs,
		}

		if !p.includeSymbol(c) {
			continue
		}

		for _, v := range vs.Values {
			var val Value

			switch vt := v.(type) {
			case *ast.BasicLit:
				val.Value = vt.Value
				val.Type = typeNames[vt.Kind]
			case *ast.CallExpr:
				if lit, ok := vt.Args[0].(*ast.BasicLit); ok {
					val.Value = lit.Value
				}

				val.Type = printNodes(vt.Fun)
				val.Specific = true
			case *ast.Ident:
				val.Type = vt.Name
			default:
				panic(fmt.Errorf("unsupported const value type %T", vt))
			}

			if vs.Type != nil {
				val.Type = printNodes(vs.Type)
				val.Specific = true
			}

			c.Values = append(c.Values, val)
		}

		cg.Consts = append(cg.Consts, c)
	}

	return cg
}

func (p *Parser) parseFuncs(pkg *Package, fns []*doc.Func) error {
	for _, fn := range fns {
		pfn := p.parseFunc(fn)
		if !p.includeSymbol(pfn) {
			continue
		}

		pkg.Funcs = append(pkg.Funcs, pfn)
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

			td := TypeDef{
				Name: t.Name,
				Doc:  p.mkDoc(t.Doc),
			}

			if err := p.parseConsts(pkg, t.Consts); err != nil {
				return fmt.Errorf("parsing consts for %s type: %w", t.Name, err)
			}

			if err := p.parseFuncs(pkg, t.Funcs); err != nil {
				return fmt.Errorf("parsing functions for %s type: %w", t.Name, err)
			}

			switch ts := typeSpec.Type.(type) {
			case *ast.Ident:
				td.Type = ts.Name
			case *ast.StructType:
				td.Type = "struct"
				td.Fields = p.parseFieldList(ts.Fields, SymbolStructField)
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
							Params:  p.parseFieldList(ft.Params, SymbolParamField),
							Results: p.parseFieldList(ft.Results, SymbolResultField),
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
				td.Params = p.parseFieldList(ts.Params, SymbolParamField)
				td.Results = p.parseFieldList(ts.Results, SymbolResultField)
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

			if !p.includeSymbol(td) {
				continue
			}

			for _, m := range t.Methods {
				pm := p.parseFunc(m)
				if !p.includeSymbol(pm) {
					continue
				}

				td.Methods = append(td.Methods, pm)
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
		fr := p.parseField(decl.Recv.List[0], SymbolReceiverField)
		fn.Receiver = &fr
	}

	if decl.Type.Params != nil && decl.Type.Params.NumFields() != 0 {
		fn.Params = p.parseFieldList(decl.Type.Params, SymbolParamField)
	}

	if decl.Type.Results != nil && decl.Type.Results.NumFields() != 0 {
		fn.Results = p.parseFieldList(decl.Type.Results, SymbolResultField)
	}

	return fn
}

func (p *Parser) parseFieldList(fl *ast.FieldList, st SymbolType) []Field {
	if fl == nil {
		return nil
	}

	res := make([]Field, 0, len(fl.List))

	for _, f := range fl.List {
		pf := p.parseField(f, st)
		if !p.includeSymbol(pf) {
			continue
		}

		res = append(res, pf)
	}

	return res
}

func (p *Parser) parseField(af *ast.Field, st SymbolType) Field {
	f := Field{
		Names:      identNames(af.Names),
		Type:       printNodes(af.Type),
		symbolType: st,
	}

	if af.Doc != nil {
		f.Doc = p.mkDoc(af.Doc.Text())
	}

	if af.Comment != nil {
		f.Comment = p.mkDoc(af.Comment.Text())
	}

	return f
}

func (p *Parser) includeSymbol(s Symbol) bool {
	for _, f := range p.filters {
		if !f.Include(s) {
			return false
		}
	}

	return true
}

func (p *Parser) mkDoc(fullDoc string) string {
	fullDoc = strings.TrimSpace(fullDoc)

	if p.noDocs {
		return ""
	}

	fullDoc = strings.TrimPrefix(strings.TrimSpace(fullDoc), "// ")

	if p.fullDocs {
		return fullDoc
	}

	pkg := doc.Package{}

	return pkg.Synopsis(fullDoc)
}

// WithFullDocs configures a [Parser] to include full doc comments instead of
// short synopsis comments.
func WithFullDocs() ParserOption {
	return &fullDocs{}
}

type fullDocs struct{}

func (*fullDocs) apply(p *Parser) error {
	p.fullDocs = true
	return nil
}

func (*fullDocs) Fingerprint() uint64 {
	h := fnv.New64a()

	h.Sum([]byte("fullDocs"))

	return h.Sum64()
}

// WithNoDocs configures a [Parser] to not include any doc comments for symbols.
func WithNoDocs() ParserOption {
	return &noDocs{}
}

type noDocs struct{}

func (*noDocs) apply(p *Parser) error {
	p.noDocs = true
	return nil
}

func (*noDocs) Fingerprint() uint64 {
	h := fnv.New64a()

	h.Sum([]byte("noDocs"))

	return h.Sum64()
}

// WithSymbolFilters configures a [Parser] to filter package symbols with
// provided filter functions.
func WithSymbolFilters(filters ...SymbolFilter) ParserOption {
	return &symbolFilters{f: filters}
}

type symbolFilters struct {
	f []SymbolFilter
}

func (sf *symbolFilters) apply(p *Parser) error {
	p.filters = sf.f
	return nil
}

func (sf *symbolFilters) Fingerprint() uint64 {
	h := fnv.New64a()

	h.Sum([]byte("symbolFilters"))

	sum := h.Sum64()

	for _, f := range sf.f {
		sum += f.Fingerprint()
	}

	return sum
}

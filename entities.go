package pkgdmp

import (
	"fmt"
	"go/format"
	"io"
	"sort"
	"strings"
)

// Identifier represents a program entity such as a function, struct, method,
// interface, etc. that can be exported by a package or kept private.
type Identifier interface {
	Ident() string
	IsExported() bool
}

// Package represents a go package containing functions and types such as
// structs and interfaces.
type Package struct {
	Name  string    `json:"name"`
	Doc   string    `json:"doc,omitempty"`
	Funcs []Func    `json:"funcs,omitempty"`
	Types []TypeDef `json:"types"`
}

// OrderIdents arranges package entities, sorting them first by whether they are
// exported or unexported, and then alphabetically.
func (p Package) OrderIdents() {
	p.orderFuncs()
}

// Source returns the formatted package signature source.
func (p Package) Source() (string, error) {
	formatted, err := format.Source([]byte(p.String()))
	if err != nil {
		return "", fmt.Errorf("formatting source: %w", err)
	}

	return string(formatted), nil
}

// String returns the unformatted package signature source.
func (p Package) String() string {
	var b strings.Builder

	if p.Doc != "" {
		b.WriteString(mkComment(p.Doc))
	}

	fmt.Fprintf(&b, "package %s", p.Name)

	for _, t := range p.Types {
		fmt.Fprintf(&b, "\n\n%s", t)
	}

	for _, f := range p.Funcs {
		fmt.Fprintf(&b, "\n\n%s", f)
	}

	b.WriteString("\n")

	return b.String()
}

func (p Package) orderFuncs() {
	sortFn := func(i, j int) bool {
		funcI := p.Funcs[i]
		funcJ := p.Funcs[j]

		return funcI.Less(funcJ)
	}

	sort.SliceStable(p.Funcs, sortFn)
}

// Func represents a function or a struct method if the Receiver field contains
// a pointer to a [FuncReceiver].
type Func struct {
	Receiver *Field  `json:"receiver,omitempty"`
	Name     string  `json:"name"`
	Doc      string  `json:"doc,omitempty"`
	Comment  string  `json:"comment,omitempty"`
	Params   []Field `json:"params,omitempty"`
	Results  []Field `json:"results,omitempty"`
	funcKw   bool
}

// Ident returns the function's name.
//
// Part of the [Identifier] interface implementation.
func (f Func) Ident() string {
	return f.Name
}

// IsExported returns true if the function is exported.
//
// Part of the [Identifier] interface implementation.
func (f Func) IsExported() bool {
	return isExportedIdent(f.Name)
}

// Less returns true if Func must sort before other Func.
func (f Func) Less(other Func) bool {
	if f.IsExported() && !other.IsExported() {
		return true
	} else if !f.IsExported() && other.IsExported() {
		return false
	}

	return f.Ident() > other.Ident()
}

// String returns the function signature code.
func (f Func) String() string {
	var b strings.Builder

	if f.Doc != "" {
		b.WriteString(mkComment(f.Doc))
	}

	if f.funcKw {
		b.WriteString("func ")
	}

	if f.Receiver != nil {
		fmt.Fprintf(&b, "(%s) ", f.Receiver)
	}

	fmt.Fprintf(&b, "%s(%s) %s", f.Name, fieldsList(f.Params), resultsList(f.Results))

	if f.Comment != "" {
		fmt.Fprintf(&b, " // %s", f.Comment)
	}

	return b.String()
}

type TypeDef struct {
	Type    string  `json:"type"`
	Name    string  `json:"name"`
	Doc     string  `json:"doc,omitempty"`
	Key     string  `json:"key,omitempty"`
	Value   string  `json:"value,omitempty"`
	Dir     string  `json:"dir,omitempty"`
	Elt     string  `json:"elt,omitempty"`
	Len     string  `json:"len,omitempty"`
	Params  []Field `json:"params,omitempty"`
	Results []Field `json:"results,omitempty"`
	Fields  []Field `json:"fields,omitempty"`
	Methods []Func  `json:"methods,omitempty"`
}

func (td TypeDef) String() string {
	var b strings.Builder

	switch td.Type {
	case "struct":
		printStructType(&b, td)
	case "interface":
		printInterfaceType(&b, td)
	case "func":
		printFuncType(&b, td)
	case "map":
		printMapType(&b, td)
	case "chan":
		printChanType(&b, td)
	case "array":
		printArrayType(&b, td)
	default:
		if td.Doc != "" {
			b.WriteString(mkComment(td.Doc))
		}

		fmt.Fprintf(&b, "type %s %s", td.Name, td.Type)

		for _, m := range td.Methods {
			fmt.Fprintf(&b, "\n\n%s", m)
		}
	}

	return b.String()
}

// Field represents a function parameter, result, or struct field.
type Field struct {
	Type    string   `json:"type"`
	Doc     string   `json:"doc"`
	Comment string   `json:"comment"`
	Names   []string `json:"names"`
}

// Ident returns the name of the struct field.
//
// Part of the [Identifier] interface implementation.
func (sf Field) Ident() string {
	return sf.Names[0]
}

// IsExported returns true if the struct field is exported.
//
// Part of the [Identifier] interface implementation.
func (sf Field) IsExported() bool {
	return isExportedIdent(sf.Names[0])
}

// String returns the unformatted struct field code fragment.
func (sf Field) String() string {
	var b strings.Builder

	if sf.Doc != "" {
		b.WriteString(mkComment(sf.Doc))
	}

	fmt.Fprintf(&b, "%s %s", strings.Join(sf.Names, ", "), sf.Type)

	if sf.Comment != "" {
		fmt.Fprintf(&b, " // %s", sf.Comment)
	}

	return b.String()
}

func printStructType(w io.Writer, s TypeDef) {
	if s.Doc != "" {
		fmt.Fprint(w, mkComment(s.Doc))
	}

	fmt.Fprintf(w, "type %s struct {", s.Name)

	if len(s.Fields) != 0 {
		fmt.Fprint(w, "\n")

		for _, f := range s.Fields {
			fmt.Fprintf(w, "    %s\n", f)
		}
	}

	fmt.Fprint(w, "}")

	if len(s.Methods) == 0 {
		return
	}

	for _, fn := range s.Methods {
		fmt.Fprintf(w, "\n\n%s", fn)
	}
}

func printInterfaceType(w io.Writer, iface TypeDef) {
	if iface.Doc != "" {
		fmt.Fprint(w, mkComment(iface.Doc))
	}

	fmt.Fprintf(w, "type %s interface {", iface.Name)

	if len(iface.Methods) != 0 {
		fmt.Fprint(w, "\n")

		for _, m := range iface.Methods {
			fmt.Fprintf(w, "    %s\n", m)
		}
	}

	fmt.Fprint(w, "}")
}

func printFuncType(w io.Writer, f TypeDef) {
	if f.Doc != "" {
		fmt.Fprint(w, mkComment(f.Doc))
	}

	fmt.Fprintf(w, "type %s func(%s) %s", f.Name, fieldsList(f.Params), resultsList(f.Results))
}

func printMapType(w io.Writer, mt TypeDef) {
	if mt.Doc != "" {
		fmt.Fprint(w, mkComment(mt.Doc))
	}

	fmt.Fprintf(w, "type %s map[%s]%s", mt.Name, mt.Key, mt.Value)

	if len(mt.Methods) == 0 {
		return
	}

	for _, m := range mt.Methods {
		fmt.Printf("\n\n%s", m)
	}
}

func printChanType(w io.Writer, ch TypeDef) {
	if ch.Doc != "" {
		fmt.Fprint(w, mkComment(ch.Doc))
	}

	fmt.Fprintf(w, "type %s ", ch.Name)

	switch ch.Dir {
	case "recv":
		fmt.Fprint(w, "<-chan ")
	case "send":
		fmt.Fprint(w, "chan<- ")
	default:
		fmt.Fprint(w, "chan ")
	}

	fmt.Fprint(w, ch.Value)
}

func printArrayType(w io.Writer, a TypeDef) {
	if a.Doc != "" {
		fmt.Fprint(w, mkComment(a.Doc))
	}

	fmt.Fprintf(w, "type %s [%s]%s", a.Name, a.Len, a.Elt)

	if len(a.Methods) == 0 {
		return
	}

	for _, m := range a.Methods {
		fmt.Printf("\n\n%s", m)
	}
}

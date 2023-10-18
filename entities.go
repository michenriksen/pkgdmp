package pkgdmp

import (
	"fmt"
	"go/format"
	"io"
	"strings"
)

// Package represents a go package containing functions and types such as
// structs and interfaces.
type Package struct {
	Name  string    `json:"name"`
	Doc   string    `json:"doc,omitempty"`
	Funcs []Func    `json:"funcs,omitempty"`
	Types []TypeDef `json:"types"`
}

// Source returns the formatted package signature source.
func (p *Package) Source() (string, error) {
	formatted, err := format.Source([]byte(p.String()))
	if err != nil {
		return "", fmt.Errorf("formatting source: %w", err)
	}

	return string(formatted), nil
}

// Print writes unformatted package code to writer.
func (p *Package) Print(w io.Writer) {
	if p.Doc != "" {
		fmt.Fprint(w, mkComment(p.Doc))
	}

	fmt.Fprintf(w, "package %s", p.Name)

	for _, t := range p.Types {
		fmt.Fprint(w, "\n\n")
		t.Print(w)
	}

	for _, f := range p.Funcs {
		fmt.Fprint(w, "\n\n")
		f.Print(w)
	}

	fmt.Fprint(w, "\n")
}

// String returns the unformatted package signature source.
func (p *Package) String() string {
	var b strings.Builder

	p.Print(&b)

	return b.String()
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
func (f Func) Ident() string {
	return f.Name
}

// IsExported returns true if the function is exported.
func (f Func) IsExported() bool {
	return isExportedIdent(f.Name)
}

// Print writes unformatted function signature code to writer.
func (f Func) Print(w io.Writer) {
	if f.Doc != "" {
		fmt.Fprint(w, mkComment(f.Doc))
	}

	if f.funcKw {
		fmt.Fprint(w, "func ")
	}

	if f.Receiver != nil {
		fmt.Fprint(w, "(")
		f.Receiver.Print(w)
		fmt.Fprint(w, ") ")
	}

	fmt.Fprintf(w, "%s(%s) %s", f.Name, fieldsList(f.Params), resultsList(f.Results))

	if f.Comment != "" {
		fmt.Fprintf(w, " // %s", f.Comment)
	}
}

// String returns the function signature code.
func (f Func) String() string {
	var b strings.Builder

	f.Print(&b)

	return b.String()
}

// TypeDef represents a type definition.
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

// Ident returns the type definition's name.
func (td TypeDef) Ident() string {
	return td.Name
}

// IsExported returns true if the type definition is exported.
func (td TypeDef) IsExported() bool {
	return isExportedIdent(td.Name)
}

// Print writes unformatted type definition code to writer.
func (td TypeDef) Print(w io.Writer) {
	switch td.Type {
	case "struct":
		printStructType(w, td)
	case "interface":
		printInterfaceType(w, td)
	case "func":
		printFuncType(w, td)
	case "map":
		printMapType(w, td)
	case "chan":
		printChanType(w, td)
	case "array":
		printArrayType(w, td)
	default:
		if td.Doc != "" {
			fmt.Fprint(w, mkComment(td.Doc))
		}

		fmt.Fprintf(w, "type %s %s", td.Name, td.Type)

		for _, m := range td.Methods {
			fmt.Fprint(w, "\n\n")
			m.Print(w)
		}
	}
}

// String returns the type definition code.
func (td TypeDef) String() string {
	var b strings.Builder

	td.Print(&b)

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
func (sf Field) Ident() string {
	return sf.Names[0]
}

// IsExported returns true if the struct field is exported.
func (sf Field) IsExported() bool {
	return isExportedIdent(sf.Names[0])
}

// Print writes unformatted field code fragment to writer.
func (sf Field) Print(w io.Writer) {
	if sf.Doc != "" {
		fmt.Fprint(w, mkComment(sf.Doc))
	}

	fmt.Fprintf(w, "%s %s", strings.Join(sf.Names, ", "), sf.Type)

	if sf.Comment != "" {
		fmt.Fprintf(w, " // %s", sf.Comment)
	}
}

// String returns the unformatted field code fragment.
func (sf Field) String() string {
	var b strings.Builder

	sf.Print(&b)

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
			f.Print(w)
			fmt.Fprint(w, "\n")
		}
	}

	fmt.Fprint(w, "}")

	if len(s.Methods) == 0 {
		return
	}

	for _, fn := range s.Methods {
		fmt.Fprint(w, "\n\n")
		fn.Print(w)
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

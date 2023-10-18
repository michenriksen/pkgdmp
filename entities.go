package pkgdmp

import (
	"fmt"
	"go/format"
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
	Name       string          `json:"name"`
	Doc        string          `json:"doc,omitempty"`
	Funcs      []Func          `json:"funcs,omitempty"`
	FuncTypes  []FuncType      `json:"funcTypes,omitempty"`
	Structs    []StructType    `json:"structs,omitempty"`
	Interfaces []InterfaceType `json:"interfaces,omitempty"`
}

// OrderIdents arranges package entities, sorting them first by whether they are
// exported or unexported, and then alphabetically.
func (p Package) OrderIdents() {
	p.orderFuncs()
	p.orderFuncTypes()
	p.orderStructs()
	p.orderInterfaces()
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

	for _, ft := range p.FuncTypes {
		fmt.Fprintf(&b, "\n\n%s", ft)
	}

	for _, s := range p.Structs {
		fmt.Fprintf(&b, "\n\n%s", s)
	}

	for _, f := range p.Funcs {
		fmt.Fprintf(&b, "\n\n%s", f)
	}

	for _, i := range p.Interfaces {
		fmt.Fprintf(&b, "\n\n%s", i)
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

func (p Package) orderFuncTypes() {
	sortFn := func(i, j int) bool {
		funcTypeI := p.FuncTypes[i]
		funcTypeJ := p.FuncTypes[j]

		return funcTypeI.Less(funcTypeJ)
	}

	sort.SliceStable(p.FuncTypes, sortFn)
}

func (p Package) orderStructs() {
	sortFn := func(i, j int) bool {
		structI := p.Structs[i]
		structJ := p.Structs[j]

		return structI.Less(structJ)
	}

	sort.SliceStable(p.Structs, sortFn)

	for _, s := range p.Structs {
		s.OrderIdents()
	}
}

func (p Package) orderInterfaces() {
	sortFn := func(i, j int) bool {
		ifaceI := p.Interfaces[i]
		ifaceJ := p.Interfaces[j]

		return ifaceI.Less(ifaceJ)
	}

	sort.SliceStable(p.Interfaces, sortFn)
}

// Func represents a function or a struct method if the Receiver field contains
// a pointer to a [FuncReceiver].
type Func struct {
	Receiver *FuncReceiver `json:"receiver,omitempty"`
	Name     string        `json:"name"`
	Doc      string        `json:"synopsis,omitempty"`
	Params   []FuncParam   `json:"params,omitempty"`
	Results  []FuncResult  `json:"results,omitempty"`
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

	fmt.Fprintf(&b, "%s(%s) %s", f.Name, paramsList(f.Params), resultsList(f.Results))

	return b.String()
}

// FuncReceiver represents a function receiver.
type FuncReceiver struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// String returns the function receiver code fragment.
func (fr FuncReceiver) String() string {
	if fr.Name == "" {
		return fr.Type
	}

	return fmt.Sprintf("%s %s", fr.Name, fr.Type)
}

// FuncParam represents a function parameter.
type FuncParam struct {
	Type  string   `json:"type"`
	Names []string `json:"names"`
}

// String returns the function parameter code fragment.
func (fp FuncParam) String() string {
	if len(fp.Names) == 0 {
		return fp.Type
	}

	return fmt.Sprintf("%s %s", strings.Join(fp.Names, ", "), fp.Type)
}

// FuncResult represents a function result (return value).
type FuncResult struct {
	Type  string   `json:"type"`
	Names []string `json:"names"`
}

// String returns the function result code fragment.
func (fr FuncResult) String() string {
	if len(fr.Names) == 0 {
		return fr.Type
	}

	return fmt.Sprintf("%s %s", strings.Join(fr.Names, ", "), fr.Type)
}

// FuncType represents a function type definition.
type FuncType struct {
	Name    string       `json:"name"`
	Doc     string       `json:"doc"`
	Params  []FuncParam  `json:"params,omitempty"`
	Results []FuncResult `json:"results,omitempty"`
}

// Ident returns the name of the function type.
//
// Part of the [Identifier] interface implementation.
func (ft FuncType) Ident() string {
	return ft.Name
}

// IsExported returns true if the function type is exported.
//
// Parts of the [Identifier] interface implementation.
func (ft FuncType) IsExported() bool {
	return isExportedIdent(ft.Name)
}

// Less returns true if FuncType must sort before other FuncType.
func (ft FuncType) Less(other FuncType) bool {
	if ft.IsExported() && !other.IsExported() {
		return true
	} else if !ft.IsExported() && other.IsExported() {
		return false
	}

	return ft.Ident() > other.Ident()
}

// String returns the function type definition code.
func (ft FuncType) String() string {
	var b strings.Builder

	if ft.Doc != "" {
		b.WriteString(mkComment(ft.Doc))
	}

	fmt.Fprintf(&b, "type %s %s(%s)", ft.Name, "func", paramsList(ft.Params))

	if len(ft.Results) == 0 {
		return b.String()
	}

	b.WriteString(" " + resultsList(ft.Results))

	return b.String()
}

// StructType represents a struct definition with fields and methods.
type StructType struct {
	Name   string        `json:"name"`
	Doc    string        `json:"doc"`
	Fields []StructField `json:"fields,omitempty"`
	Funcs  []Func        `json:"funcs,omitempty"`
}

// Ident returns the name of the struct type.
//
// Part of the [Identifier] interface implementation.
func (s StructType) Ident() string {
	return s.Name
}

// IsExported returns true if the struct type is exported.
//
// Part of the [Identifier] interface implementation.
func (s StructType) IsExported() bool {
	return isExportedIdent(s.Name)
}

// Less returns true if Struct must sort before other Struct.
func (s StructType) Less(other StructType) bool {
	if s.IsExported() && !other.IsExported() {
		return true
	} else if !s.IsExported() && other.IsExported() {
		return false
	}

	return s.Ident() > other.Ident()
}

// OrderIdents arranges struct methods, sorting them first by whether they are
// exported or unexported, and then alphabetically.
func (s StructType) OrderIdents() {
	sortFn := func(i, j int) bool {
		funcI := s.Funcs[i]
		funcJ := s.Funcs[j]

		return funcI.Less(funcJ)
	}

	sort.SliceStable(s.Funcs, sortFn)
}

// String returns the unformatted struct type definition and method signature
// code.
func (s StructType) String() string {
	var b strings.Builder

	if s.Doc != "" {
		b.WriteString(mkComment(s.Doc))
	}

	fmt.Fprintf(&b, "type %s struct {", s.Name)

	if len(s.Fields) != 0 {
		b.WriteRune('\n')

		for _, f := range s.Fields {
			fmt.Fprintf(&b, "    %s\n", f)
		}
	}

	b.WriteRune('}')

	if len(s.Funcs) == 0 {
		return b.String()
	}

	for _, fn := range s.Funcs {
		fmt.Fprintf(&b, "\n\n%s", fn)
	}

	return b.String()
}

// StructField represents a [StructType] field.
type StructField struct {
	Type    string   `json:"type"`
	Doc     string   `json:"doc"`
	Comment string   `json:"comment"`
	Names   []string `json:"names"`
}

// Ident returns the name of the struct field.
//
// Part of the [Identifier] interface implementation.
func (sf StructField) Ident() string {
	return sf.Names[0]
}

// IsExported returns true if the struct field is exported.
//
// Part of the [Identifier] interface implementation.
func (sf StructField) IsExported() bool {
	return isExportedIdent(sf.Names[0])
}

// String returns the unformatted struct field code fragment.
func (sf StructField) String() string {
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

// InterfaceType represents an interface definition.
type InterfaceType struct {
	Name  string `json:"name"`
	Doc   string `json:"doc"`
	Funcs []Func `json:"funcs,omitempty"`
}

// Ident returns the name of the interface.
//
// Part of the [Identifier] interface implementation.
func (i InterfaceType) Ident() string {
	return i.Name
}

// IsExported returns true if the interface is exported.
//
// Part of the [Identifier] interface implementation.
func (i InterfaceType) IsExported() bool {
	return isExportedIdent(i.Name)
}

// Less returns true if Interface must sort before other Interface.
func (i InterfaceType) Less(other InterfaceType) bool {
	if i.IsExported() && !other.IsExported() {
		return true
	} else if !i.IsExported() && other.IsExported() {
		return false
	}

	return i.Ident() > other.Ident()
}

// String returns the interface's unformatted definition code.
func (i InterfaceType) String() string {
	var b strings.Builder

	if i.Doc != "" {
		b.WriteString(mkComment(i.Doc))
	}

	fmt.Fprintf(&b, "type %s interface {", i.Name)

	if len(i.Funcs) == 0 {
		b.WriteString("}")
		return b.String()
	}

	b.WriteString("\n")

	for _, f := range i.Funcs {
		fmt.Fprintf(&b, "    %s\n", f)
	}

	b.WriteString("}")

	return b.String()
}

package pkgdmp

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// FilterAction configures a [SymbolFilterFn].
type FilterAction int

const (
	Exclude FilterAction = iota // Exclude symbols.
	Include                     // Include symbols.
)

// String returns a string representation of a filter action.
func (fa FilterAction) String() string {
	return [...]string{
		"Exclude",
		"Include",
	}[fa]
}

func (fa FilterAction) GoString() string {
	return "pkgdmp." + fa.String()
}

// SymbolType represents a type of package symbol.
type SymbolType int

const (
	SymbolUnknown       SymbolType = iota
	SymbolPackage                  // `package mypackage`
	SymbolConst                    // `const myConst = ...`
	SymbolIdentType                // `type MyInt int`
	SymbolFuncType                 // `type MyFunc func(...)`
	SymbolStructType               // `type MyStruct { ... }`
	SymbolInterfaceType            // `type MyInterface { ... }`
	SymbolMapType                  // `type MyMap map[...]...`
	SymbolChanType                 // `type MyChan chan ...`
	SymbolArrayType                // `type MyArray []string`
	SymbolFunc                     // `func MyFunc(...) { ... }`
	SymbolStructField              // Struct field.
	SymbolParamField               // Function parameter field.
	SymbolResultField              // Function result field.
	SymbolReceiverField            // Function Receiver field.
)

// unfilterableMap contains symbol types that filter functions should always
// return true for.
var unfilterableMap = map[SymbolType]struct{}{
	SymbolPackage:       {},
	SymbolParamField:    {},
	SymbolResultField:   {},
	SymbolReceiverField: {},
}

// String returns a string representation of a symbol type.
func (st SymbolType) String() string {
	return [...]string{
		"SymbolUnknown",
		"SymbolPackage",
		"SymbolConst",
		"SymbolIdentType",
		"SymbolFunctionType",
		"SymbolStructType",
		"SymbolInterfaceType",
		"SymbolMapType",
		"SymbolChanType",
		"SymbolArrayType",
		"SymbolFunc",
		"SymbolStructField",
		"SymbolParamField",
		"SymbolResultField",
		"SymbolReceiverField",
	}[st]
}

func (st SymbolType) GoString() string {
	return "pkgdmp." + st.String()
}

// Symbol represents a symbol such as a const, type definition, or function.
type Symbol interface {
	Ident() string
	IsExported() bool
	SymbolType() SymbolType
}

// SymbolFilter filters symbols by different conditions.
type SymbolFilter interface {
	// Include should return true if symbol should be included according to
	// the filter's logic and configuration.
	Include(Symbol) bool

	// String should return a string representation of the filter.
	//
	// This method is mainly intended for testing purposes.
	String() string
}

// FilterUnexported creates a filter that determines whether to include or
// exclude unexported symbols.
func FilterUnexported(action FilterAction) SymbolFilter {
	return &filterUnexported{action: action}
}

type filterUnexported struct {
	action FilterAction
}

func (f *filterUnexported) Include(s Symbol) bool {
	if isUnfilterable(s) {
		return true
	}

	return f.action == Include || s.IsExported()
}

func (f *filterUnexported) String() string {
	return fmt.Sprintf("filterUnexported(action=%s)", f.action)
}

// FilterSymbolTypes creates a filter function that determines whether to
// include or exclude symbols of different types.
func FilterSymbolTypes(action FilterAction, types ...SymbolType) SymbolFilter {
	stMap := make(map[SymbolType]struct{}, len(types))

	for _, t := range types {
		stMap[t] = struct{}{}
	}

	return &filterSymbolTypes{
		stMap:  stMap,
		action: action,
	}
}

type filterSymbolTypes struct {
	stMap  map[SymbolType]struct{}
	action FilterAction
}

func (f *filterSymbolTypes) Include(s Symbol) bool {
	if isUnfilterable(s) {
		return true
	}

	_, ok := f.stMap[s.SymbolType()]

	if f.action == Include {
		return ok
	}

	return !ok
}

func (f *filterSymbolTypes) String() string {
	sts := make([]string, 0, len(f.stMap))

	for st := range f.stMap {
		sts = append(sts, st.String())
	}

	sort.Strings(sts)

	return fmt.Sprintf("filterSymbolTypes(action=%s,symbolTypes=%s)", f.action, strings.Join(sts, ","))
}

// FilterSymbolTypes creates a filter function that determines whether to
// include or exclude symbols with matching idents.
func FilterMatchingIdents(action FilterAction, p *regexp.Regexp) SymbolFilter {
	return &filterMatchingIdents{action: action, pattern: p}
}

type filterMatchingIdents struct {
	pattern *regexp.Regexp
	action  FilterAction
}

func (f *filterMatchingIdents) Include(s Symbol) bool {
	if isUnfilterable(s) {
		return true
	}

	match := f.pattern.MatchString(s.Ident())

	if f.action == Include {
		return match
	}

	return !match
}

func (f *filterMatchingIdents) String() string {
	return fmt.Sprintf("filterMatchingIdents(action=%s,pattern=%s)", f.action, f.pattern)
}

// FilterPackages creates a filter function that determines whether to include
// or exclude packages matching provided names.
func FilterPackages(action FilterAction, names ...string) SymbolFilter {
	pkgMap := make(map[string]struct{}, len(names))

	for _, n := range names {
		pkgMap[n] = struct{}{}
	}

	return &filterPackages{pkgMap: pkgMap, action: action}
}

type filterPackages struct {
	pkgMap map[string]struct{}
	action FilterAction
}

func (f *filterPackages) Include(s Symbol) bool {
	if s.SymbolType() != SymbolPackage {
		return true
	}

	_, ok := f.pkgMap[s.Ident()]

	if f.action == Include {
		return ok
	}

	return !ok
}

func (f *filterPackages) String() string {
	names := make([]string, 0, len(f.pkgMap))

	for n := range f.pkgMap {
		names = append(names, n)
	}

	sort.Strings(names)

	return fmt.Sprintf("filterPackages(action=%s,names=%s)", f.action, strings.Join(names, ","))
}

func isUnfilterable(s Symbol) bool {
	if _, ok := unfilterableMap[s.SymbolType()]; ok {
		return true
	}

	return false
}

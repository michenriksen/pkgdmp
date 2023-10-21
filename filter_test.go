package pkgdmp_test

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"testing"

	"github.com/michenriksen/pkgdmp"
)

var symbolTypes = []pkgdmp.SymbolType{
	pkgdmp.SymbolConst,
	pkgdmp.SymbolIdentType,
	pkgdmp.SymbolFuncType,
	pkgdmp.SymbolStructType,
	pkgdmp.SymbolInterfaceType,
	pkgdmp.SymbolMapType,
	pkgdmp.SymbolChanType,
	pkgdmp.SymbolArrayType,
	pkgdmp.SymbolFunc,
	pkgdmp.SymbolMethod,
}

func TestFilterUnexported(t *testing.T) {
	exported := newSymbol(t, "MyExported", randSymbolType(t))
	unexported := newSymbol(t, "myUnexported", randSymbolType(t))
	exportedSField := newSymbol(t, "MyExported", pkgdmp.SymbolStructField)
	unexportedSField := newSymbol(t, "myUnexported", pkgdmp.SymbolStructField)

	tt := []struct {
		s      pkgdmp.Symbol
		action pkgdmp.FilterAction
		want   bool
	}{
		{exported, pkgdmp.Include, true},
		{exported, pkgdmp.Exclude, true},
		{unexported, pkgdmp.Include, true},
		{unexported, pkgdmp.Exclude, false},
		{exportedSField, pkgdmp.Include, true},
		{exportedSField, pkgdmp.Exclude, true},
		{unexportedSField, pkgdmp.Include, true},
		{unexportedSField, pkgdmp.Exclude, false},
	}

	for _, tc := range tt {
		tc := tc

		name := fmt.Sprintf("returns %t for %s when action is %s",
			tc.want, tc.s.Ident(), tc.action,
		)

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := pkgdmp.FilterUnexported(tc.action)

			if f.Include(tc.s) == tc.want {
				return
			}

			t.Errorf("expected FilterUnexported(%v) to return %t for %s",
				tc.action, tc.want, tc.s,
			)
		})
	}
}

func TestFilterSymbolTypes(t *testing.T) {
	tt := []pkgdmp.Symbol{
		newSymbol(t, "myConst", pkgdmp.SymbolConst),
		newSymbol(t, "MyCustomType", pkgdmp.SymbolIdentType),
		newSymbol(t, "myFuncType", pkgdmp.SymbolFuncType),
		newSymbol(t, "MyStruct", pkgdmp.SymbolStructType),
		newSymbol(t, "myInterface", pkgdmp.SymbolInterfaceType),
		newSymbol(t, "MyMap", pkgdmp.SymbolMapType),
		newSymbol(t, "myChan", pkgdmp.SymbolChanType),
		newSymbol(t, "MyArray", pkgdmp.SymbolArrayType),
		newSymbol(t, "MyFunc", pkgdmp.SymbolFunc),
		newSymbol(t, "MyMethod", pkgdmp.SymbolMethod),
	}

	t.Run("returns true when all symbol types are included", func(t *testing.T) {
		t.Parallel()

		f := pkgdmp.FilterSymbolTypes(pkgdmp.Include, symbolTypes...)

		for _, tc := range tt {
			if f.Include(tc) == true {
				continue
			}

			t.Errorf("expected FilterSymbolTypes(pkgdmp.Include, %s) to return true for %s",
				symbolTypesList(t, symbolTypes...), tc,
			)
		}
	})

	t.Run("returns false when all symbol types are excluded", func(t *testing.T) {
		t.Parallel()

		f := pkgdmp.FilterSymbolTypes(pkgdmp.Exclude, symbolTypes...)

		for _, tc := range tt {
			if f.Include(tc) == false {
				continue
			}

			t.Errorf("expected FilterSymbolTypes(pkgdmp.Exclude, %s) to return true for %s",
				symbolTypesList(t, symbolTypes...), tc,
			)
		}
	})

	t.Run("returns false when include is true and symbol type is not included", func(t *testing.T) {
		t.Parallel()

		for _, tc := range tt {
			st := symbolTypesExcept(t, tc.SymbolType())
			f := pkgdmp.FilterSymbolTypes(pkgdmp.Include, st...)

			if f.Include(tc) == false {
				continue
			}

			t.Errorf("expected FilterSymbolTypes(pkgdmp.Include, %s) to return false for %s",
				symbolTypesList(t, st...), tc,
			)
		}
	})

	t.Run("returns true when include is false and symbol type is not included", func(t *testing.T) {
		t.Parallel()

		for _, tc := range tt {
			st := symbolTypesExcept(t, tc.SymbolType())
			f := pkgdmp.FilterSymbolTypes(pkgdmp.Exclude, st...)

			if f.Include(tc) == true {
				continue
			}

			t.Errorf("expected FilterSymbolTypes(pkgdmp.Exclude, %s) to return true for %s",
				symbolTypesList(t, st...), tc,
			)
		}
	})
}

func TestFilterMatchingIdents(t *testing.T) {
	tt := []struct {
		s      pkgdmp.Symbol
		p      *regexp.Regexp
		action pkgdmp.FilterAction
		want   bool
	}{
		{newSymbol(t, "FooBar", randSymbolType(t)), regexp.MustCompile(`^FooBa(r|z)`), pkgdmp.Include, true},
		{newSymbol(t, "FooBar", randSymbolType(t)), regexp.MustCompile(`^FooBa(r|z)`), pkgdmp.Exclude, false},
		{newSymbol(t, "FooBar", randSymbolType(t)), regexp.MustCompile(`^MySymbol`), pkgdmp.Exclude, true},
		{newSymbol(t, "FooBar", randSymbolType(t)), regexp.MustCompile(`^MySymbol`), pkgdmp.Include, false},
	}

	for _, tc := range tt {
		tc := tc

		name := fmt.Sprintf("returns %t for %s with action %s and pattern %s",
			tc.want, tc.s, tc.action, tc.p,
		)

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := pkgdmp.FilterMatchingIdents(tc.action, tc.p)

			if f.Include(tc.s) == tc.want {
				return
			}

			t.Errorf("expected FilterMatchingIdents(%v, `%s`) to return %t for %s",
				tc.action, tc.p, tc.want, tc.s,
			)
		})
	}
}

type stubSymbol struct {
	ident string
	st    pkgdmp.SymbolType
}

func newSymbol(tb testing.TB, ident string, st pkgdmp.SymbolType) stubSymbol {
	tb.Helper()

	return stubSymbol{ident: ident, st: st}
}

func (ss stubSymbol) Ident() string {
	return ss.ident
}

func (ss stubSymbol) IsExported() bool {
	return strings.ToUpper(ss.ident[:1]) == ss.ident[:1]
}

func (ss stubSymbol) SymbolType() pkgdmp.SymbolType {
	return ss.st
}

func (ss stubSymbol) String() string {
	return fmt.Sprintf("%s %s", ss.st, ss.ident)
}

// randSymbolType returns a random symbol type.
func randSymbolType(tb testing.TB) pkgdmp.SymbolType {
	tb.Helper()

	ri := rand.Intn(len(symbolTypes))

	return symbolTypes[ri]
}

// symbolTypesList returns a string of comma-separated symbol types.
func symbolTypesList(tb testing.TB, st ...pkgdmp.SymbolType) string {
	tb.Helper()

	res := make([]string, len(st))

	for i, t := range st {
		res[i] = t.GoString()
	}

	return strings.Join(res, ", ")
}

// symbolTypesExcept returns all symbol types except for the ones to exclude.
func symbolTypesExcept(tb testing.TB, exclude ...pkgdmp.SymbolType) []pkgdmp.SymbolType {
	tb.Helper()

	removeMap := make(map[pkgdmp.SymbolType]struct{})
	for _, s := range exclude {
		removeMap[s] = struct{}{}
	}

	var result []pkgdmp.SymbolType

	for _, t := range symbolTypes {
		if _, ok := removeMap[t]; !ok {
			result = append(result, t)
		}
	}

	return result
}

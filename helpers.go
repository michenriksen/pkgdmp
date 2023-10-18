package pkgdmp

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
)

func identNames(idents []*ast.Ident) []string {
	iLen := len(idents)
	if iLen == 0 {
		return nil
	}

	res := make([]string, iLen)

	for i, ident := range idents {
		res[i] = ident.Name
	}

	return res
}

func isExportedIdent(name string) bool {
	return strings.ToUpper(name[:1]) == name[:1]
}

func mkComment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	var b strings.Builder

	lines := strings.Split(s, "\n")

	if len(lines) > 1 {
		for _, line := range lines {
			fmt.Fprintf(&b, "// %s\n", line)
		}

		return b.String()
	}

	lineLen, _ := fmt.Fprintf(&b, "// ")
	words := strings.Fields(s)

	for _, word := range words {
		wLen := len(word)
		if lineLen+wLen+1 < 80 {
			n, _ := fmt.Fprintf(&b, "%s ", word)
			lineLen += n

			continue
		}

		lineLen, _ = fmt.Fprintf(&b, "\n// %s ", word)
	}

	b.WriteRune('\n')

	return b.String()
}

func fieldsList(fl []Field) string {
	fLen := len(fl)
	if fLen == 0 {
		return ""
	}

	res := make([]string, fLen)

	for i, f := range fl {
		res[i] = f.String()
	}

	return strings.Join(res, ", ")
}

func resultsList(fl []Field) string {
	s := fieldsList(fl)

	if len(fl) > 1 {
		return fmt.Sprintf("(%s)", s)
	}

	return s
}

func printNodes(nodes any) string {
	var b strings.Builder

	fset := token.NewFileSet()

	printer.Fprint(&b, fset, nodes)

	return b.String()
}

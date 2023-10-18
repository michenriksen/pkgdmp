package pkgdmp

import (
	"fmt"
	"go/ast"
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

func paramsList(params []FuncParam) string {
	res := make([]string, len(params))

	for i, p := range params {
		res[i] = p.String()
	}

	return strings.Join(res, ", ")
}

func resultsList(results []FuncResult) string {
	rLen := len(results)
	if rLen == 0 {
		return ""
	}

	res := make([]string, rLen)

	for i, r := range results {
		res[i] = r.String()
	}

	if rLen == 1 && len(results[0].Names) == 0 {
		return res[0]
	}

	return fmt.Sprintf("(%s)", strings.Join(res, ", "))
}

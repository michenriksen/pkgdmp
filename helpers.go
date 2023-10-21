package pkgdmp

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"regexp"
	"strings"
)

var fieldSTMap = map[SymbolType]struct{}{
	SymbolStructField:   {},
	SymbolParamField:    {},
	SymbolResultField:   {},
	SymbolReceiverField: {},
}

var fieldTagRegexp = regexp.MustCompile(`(\w+):"(.*?)"`)

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

func isFieldSymbolType(st SymbolType) bool {
	_, ok := fieldSTMap[st]
	return ok
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

	return strings.TrimSpace(b.String())
}

func parseFieldTags(s string) [][]string {
	s = strings.Trim(s, "`")

	matches := fieldTagRegexp.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return nil
	}

	tags := make([][]string, 0, len(matches))

	for _, m := range matches {
		name := m[1]
		values := strings.Split(m[2], ",")
		tag := append([]string{name}, values...)

		tags = append(tags, tag)
	}

	return tags
}

package impl

import (
	"go/ast"
	"go/parser"
)

// Get some ordinal ast.Ident.Name from a given ast.Node. A negative will return
// the last identifier in the tree.
func getIdent(node ast.Node, ord int) string {
	ident := ""
	n := 0

	ast.Inspect(node, func(child ast.Node) bool {
		if child, ok := child.(*ast.Ident); ok {
			ident = child.Name
			n++
		}
		return ord < 0 || n == ord
	})

	return ident
}

// As a shortcut we parse the receiver expression, then just take the last
// identifier specified in the resulting ast
func getType(recv string) (string, error) {
	a, err := parser.ParseExpr(recv)
	if err != nil {
		return "", err
	}

	return getIdent(a, -1), nil
}

func getMethods(id string, f *ast.File) []*ast.FuncDecl {
	decls := []*ast.FuncDecl{}

	for _, decl := range f.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			if decl.Recv != nil && decl.Name != nil {
				for _, r := range decl.Recv.List {
					name := getIdent(r.Type, -1)
					if name == id {
						decls = append(decls, decl)
					}
				}
			}
		}
	}

	return decls
}

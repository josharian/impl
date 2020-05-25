package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// implementedFuncs returns list of Func which already implemented.
func implementedFuncs(fns []Func, recv string, srcDir string) (map[string]bool, error) {

	// determine name of receiver type
	recvType := getReceiverType(recv)

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, srcDir, nil, 0)
	if err != nil {
		return nil, err
	}

	implemented := make(map[string]bool)

	// getReceiver returns title of struct to which belongs the method
	getReceiver := func(mf *ast.FuncDecl) string {
		if mf.Recv == nil {
			return ""
		}

		for _, v := range mf.Recv.List {
			switch xv := v.Type.(type) {
			case *ast.StarExpr:
				if si, ok := xv.X.(*ast.Ident); ok {
					return si.Name
				}
			case *ast.Ident:
				return xv.Name
			}
		}

		return ""
	}

	// Convert fns to a map, to prevent accidental quadratic behavior.
	want := make(map[string]bool)
	for _, fn := range fns {
		want[fn.Name] = true
	}

	// finder is a walker func which will be called for each element in the source code of package
	// but we are interested in funcs only with receiver same to typeTitle
	finder := func(n ast.Node) bool {
		x, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		if getReceiver(x) != recvType {
			return true
		}
		name := x.Name.String()
		if want[name] {
			implemented[name] = true
		}
		return true
	}

	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			ast.Inspect(f, finder)
		}
	}

	return implemented, nil
}

// getReceiverType returns type name of receiver or fatal if receiver is invalid.
// ex: for definition "r *SomeType" will return "SomeType"
func getReceiverType(recv string) string {
	var recvType string

	// VSCode adds a trailing space to receiver (it runs impl like: impl 'r *Receiver ' io.Writer)
	// so we have to remove spaces.
	recv = strings.TrimSpace(recv)
	parts := strings.Split(recv, " ")
	switch len(parts) {
	case 1: // (SomeType)
		recvType = parts[0]
	case 2: // (x SomeType)
		recvType = parts[1]
	default:
		fatal(fmt.Sprintf("invalid receiver: %q", recv))
	}

	// Pointer to receiver should be removed too for comparison purpose.
	// But don't worry definition of default receiver won't be changed.
	return strings.TrimPrefix(recvType, "*")
}

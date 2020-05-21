package main

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// implemented checks method already implemetned.
// ifns could be requested by calling ifuncs().
func implemented(ifns []Func, fnName string) bool {
	for _, ifn := range ifns {
		if fnName == ifn.Name {
			return true
		}
	}
	return false
}

// ifuncs returns list of Func which already implemented.
func ifuncs(fns []Func, typeTitle string, srcDir string) ([]Func, error) {

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, srcDir, nil, 0)
	if err != nil {
		return nil, err
	}

	var ifns []Func

	// getReceiver returns title of struct to which belongs the method
	getReceiver := func(mf *ast.FuncDecl) string {
		if mf.Recv != nil {
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
		}
		return ""
	}

	// finder is a walker func which will be called for each element in the source code of package
	// but we are interested in funcs only with receiver same to typeTitle
	finder := func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if getReceiver(x) == typeTitle {
				for _, fn := range fns {
					if fn.Name == x.Name.String() {
						ifns = append(ifns, fn)
						break
					}
				}
			}
		}
		return true
	}

	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			ast.Inspect(f, finder)
		}
	}

	return ifns, nil
}

// impl generates method stubs for implementing an interface.
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"code.google.com/p/go.tools/imports"
)

const usage = `impl <recv> <iface>

impl generates method stubs for recv to implement iface.

Examples:

impl 'f *File' io.Reader
impl Murmur hash.Hash

Don't forget the single quotes around the receiver type
to prevent shell globbing.
`

// findInterface returns the import path and identifier of an interface.
// For example, given "http.ResponseWriter", findInterface returns
// "net/http", "ResponseWriter".
func findInterface(iface string) (path string, id string, err error) {
	// Let goimports do the heavy lifting.
	src := "package hack\nvar i " + iface + "\n"

	imp, err := imports.Process("", []byte(src), nil)
	if err != nil {
		return "", "", fmt.Errorf("couldn't parse interface: %s", iface)
	}

	// imp should now contain an appropriate import.
	// Parse out the import and the identifier.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", imp, 0)
	if err != nil {
		panic(err)
	}
	if len(f.Imports) == 0 {
		return "", "", fmt.Errorf("unrecognized interface: %s", iface)
	}
	path, err = strconv.Unquote(f.Imports[0].Path.Value)
	if err != nil {
		panic(err)
	}
	varDecl := f.Decls[1].(*ast.GenDecl)                              // var i io.Reader
	sel := varDecl.Specs[0].(*ast.ValueSpec).Type.(*ast.SelectorExpr) // io.Reader
	id = sel.Sel.Name                                                 // Reader
	return path, id, nil
}

// interfaceDecl locates the declaration for interface id in pkg, if present.
func interfaceDecl(pkg *build.Package, id string) (*token.FileSet, *ast.InterfaceType, bool) {
	fset := token.NewFileSet()
	for _, file := range pkg.GoFiles {
		f, err := parser.ParseFile(fset, filepath.Join(pkg.Dir, file), nil, 0)
		if err != nil {
			continue
		}

		for _, decl := range f.Decls {
			decl, ok := decl.(*ast.GenDecl)
			if !ok || decl.Tok != token.TYPE || len(decl.Specs) != 1 {
				continue
			}
			spec := decl.Specs[0].(*ast.TypeSpec)
			if spec.Name.Name != id {
				continue
			}
			if typ, ok := spec.Type.(*ast.InterfaceType); ok {
				return fset, typ, true
			}
		}
	}

	return nil, nil, false
}

// Pkg is a parsed build.Package.
type Pkg struct {
	*build.Package
	*token.FileSet
}

// gofmt pretty-prints n.
func (p Pkg) gofmt(n ast.Expr) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, p.FileSet, n)
	return buf.String()
}

// fullType returns the fully qualified type of n.
// Examples, assuming package net/http:
// 	fullType(int) => "int"
// 	fullType(Handler) => "http.Handler"
// 	fullType(io.Reader) => "io.Reader"
// 	fullType(*Request) => "*http.Request"
func (p Pkg) fullType(n ast.Expr) string {
	switch n := n.(type) {
	case *ast.StarExpr:
		return "*" + p.fullType(n.X)
	case *ast.Ident:
		if n.IsExported() {
			return p.Package.Name + "." + p.gofmt(n)
		}
	}
	return p.gofmt(n)
}

func (p Pkg) param(field *ast.Field) Param {
	name := ""
	if len(field.Names) > 0 {
		name = field.Names[0].Name
	}
	return Param{Name: name, Type: p.fullType(field.Type)}
}

// Method represents a method signature.
type Method struct {
	Recv string
	Func
}

// Func represents a function signature.
type Func struct {
	Name   string
	Params []Param
	Res    []Param
}

// Param represents a parameter in a function or method signature.
type Param struct {
	Name string
	Type string
}

// funcs returns the set of methods required to implement iface.
// It is called funcs rather than methods because the
// function descriptions are functions; there is no receiver.
func funcs(iface string) ([]Func, error) {
	// Locate the interface.
	path, id, err := findInterface(iface)
	if err != nil {
		return nil, err
	}

	// Locate the package containing the interface.
	bpkg, err := build.Import(path, "", 0)
	if err != nil {
		return nil, fmt.Errorf("couldn't find package %s: %v", path, err)
	}

	// Find the declaration of the interface.
	fset, decl, ok := interfaceDecl(bpkg, id)
	if !ok {
		return nil, fmt.Errorf("interface not found: %s", iface)
	}

	pkg := Pkg{FileSet: fset, Package: bpkg}

	var fns []Func
	for _, fndecl := range decl.Methods.List {
		// Handle embedded interfaces.
		if len(fndecl.Names) == 0 {
			embedded, err := funcs(pkg.fullType(fndecl.Type))
			if err != nil {
				return nil, err
			}
			fns = append(fns, embedded...)
			continue
		}

		// Extract function signatures.
		fn := Func{Name: fndecl.Names[0].Name}
		typ := fndecl.Type.(*ast.FuncType)
		if typ.Params != nil {
			for _, field := range typ.Params.List {
				fn.Params = append(fn.Params, pkg.param(field))
			}
		}
		if typ.Results != nil {
			for _, field := range typ.Results.List {
				fn.Res = append(fn.Res, pkg.param(field))
			}
		}
		fns = append(fns, fn)
	}
	return fns, nil
}

const stub = "func ({{.Recv}}) {{.Name}}" +
	"({{range .Params}}{{.Name}} {{.Type}}, {{end}})" +
	"({{range .Res}}{{.Name}} {{.Type}}, {{end}})" +
	"{\n}\n"

var tmpl = template.Must(template.New("test").Parse(stub))

// fprintStubs prints nicely formatted method stubs
// for fns using receiver expression recv.
func fprintStubs(w io.Writer, recv string, fns []Func) error {
	pr, pw := io.Pipe()

	// Print crudely, but with an easy to write template.
	go func() {
		for _, fn := range fns {
			meth := Method{Recv: recv, Func: fn}
			tmpl.Execute(pw, meth)
		}
		pw.Close()
	}()

	// Parse the crude printing.
	r := io.MultiReader(strings.NewReader("package hack\n"), pr)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", r, 0)
	if err != nil {
		panic(err)
	}

	// Print only the method declarations.
	for _, decl := range f.Decls {
		if err := printer.Fprint(w, fset, decl); err != nil {
			return err
		}
		if _, err := fmt.Fprint(w, "\n\n"); err != nil {
			return err
		}
	}

	return nil
}

// validReceiver reports whether recv is a valid receiver expression.
func validReceiver(recv string) bool {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "", "package hack\nfunc ("+recv+") Foo()", 0)
	return err == nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	recv, iface := os.Args[1], os.Args[2]
	if !validReceiver(recv) {
		fatal(fmt.Sprintf("invalid receiver: %q", recv))
	}

	fns, err := funcs(iface)
	if err != nil {
		fatal(err)
	}

	fprintStubs(os.Stdout, recv, fns)
}

func fatal(msg interface{}) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

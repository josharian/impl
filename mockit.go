package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"golang.org/x/tools/imports"
)

var (
	flagSrcDir = flag.String("dir", "", "package source directory, useful for vendored code")
)

// findInterface returns the import path and identifier of an interface.
// For example, given "http.ResponseWriter", findInterface returns
// "net/http", "ResponseWriter".
// If a fully qualified interface is given, such as "net/http.ResponseWriter",
// it simply parses the input.
// If an unqualified interface such as "UserDefinedInterface" is given, then
// the interface definition is presumed to be in the package within srcDir and
// findInterface returns "", "UserDefinedInterface".
func findInterface(iface string, srcDir string) (path string, id string, err error) {
	if len(strings.Fields(iface)) != 1 {
		return "", "", fmt.Errorf("couldn't parse interface: %s", iface)
	}

	srcPath := filepath.Join(srcDir, "__go_impl__.go")

	if slash := strings.LastIndex(iface, "/"); slash > -1 {
		// package path provided
		dot := strings.LastIndex(iface, ".")
		// make sure iface does not end with "/" (e.g. reject net/http/)
		if slash+1 == len(iface) {
			return "", "", fmt.Errorf("interface name cannot end with a '/' character: %s", iface)
		}
		// make sure iface does not end with "." (e.g. reject net/http.)
		if dot+1 == len(iface) {
			return "", "", fmt.Errorf("interface name cannot end with a '.' character: %s", iface)
		}
		// make sure iface has exactly one "." after "/" (e.g. reject net/http/httputil)
		if strings.Count(iface[slash:], ".") != 1 {
			return "", "", fmt.Errorf("invalid interface name: %s", iface)
		}
		return iface[:dot], iface[dot+1:], nil
	}

	src := []byte("package hack\n" + "var i " + iface)
	// If we couldn't determine the import path, goimports will
	// auto fix the import path.
	imp, err := imports.Process(srcPath, src, nil)
	if err != nil {
		return "", "", fmt.Errorf("couldn't parse interface: %s", iface)
	}

	// imp should now contain an appropriate import.
	// Parse out the import and the identifier.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, srcPath, imp, 0)
	if err != nil {
		panic(err)
	}

	qualified := strings.Contains(iface, ".")

	if len(f.Imports) == 0 && qualified {
		return "", "", fmt.Errorf("unrecognized interface: %s", iface)
	}

	if !qualified {
		// If !qualified, the code looks like:
		//
		// package hack
		//
		// var i Reader
		decl := f.Decls[0].(*ast.GenDecl)      // var i io.Reader
		spec := decl.Specs[0].(*ast.ValueSpec) // i io.Reader
		sel := spec.Type.(*ast.Ident)
		id = sel.Name // Reader

		return path, id, nil
	}

	// If qualified, the code looks like:
	//
	// package hack
	//
	// import (
	//   "io"
	// )
	//
	// var i io.Reader
	raw := f.Imports[0].Path.Value   // "io"
	path, err = strconv.Unquote(raw) // io
	if err != nil {
		panic(err)
	}
	decl := f.Decls[1].(*ast.GenDecl)      // var i io.Reader
	spec := decl.Specs[0].(*ast.ValueSpec) // i io.Reader
	sel := spec.Type.(*ast.SelectorExpr)   // io.Reader
	id = sel.Sel.Name                      // Reader

	return path, id, nil
}

// Pkg is a parsed build.Package.
type Pkg struct {
	*build.Package
	*token.FileSet
}

// Spec is ast.TypeSpec with the associated comment map.
type Spec struct {
	*ast.TypeSpec
	ast.CommentMap
}

// typeSpec locates the *ast.TypeSpec for type id in the import path.
func typeSpec(path string, id string, srcDir string) (Pkg, Spec, error) {
	var pkg *build.Package
	var err error

	if path == "" {
		pkg, err = build.ImportDir(srcDir, 0)
		if err != nil {
			return Pkg{}, Spec{}, fmt.Errorf("couldn't find package in %s: %v", srcDir, err)
		}
	} else {
		pkg, err = build.Import(path, srcDir, 0)
		if err != nil {
			return Pkg{}, Spec{}, fmt.Errorf("couldn't find package %s: %v", path, err)
		}
	}

	fset := token.NewFileSet() // share one fset across the whole package
	var files []string
	files = append(files, pkg.GoFiles...)
	files = append(files, pkg.CgoFiles...)
	for _, file := range files {
		f, err := parser.ParseFile(fset, filepath.Join(pkg.Dir, file), nil, parser.ParseComments)
		if err != nil {
			continue
		}

		cmap := ast.NewCommentMap(fset, f, f.Comments)

		for _, decl := range f.Decls {
			decl, ok := decl.(*ast.GenDecl)
			if !ok || decl.Tok != token.TYPE {
				continue
			}
			for _, spec := range decl.Specs {
				spec := spec.(*ast.TypeSpec)
				if spec.Name.Name != id {
					continue
				}
				p := Pkg{Package: pkg, FileSet: fset}
				s := Spec{TypeSpec: spec, CommentMap: cmap.Filter(decl)}
				return p, s, nil
			}
		}
	}
	return Pkg{}, Spec{}, fmt.Errorf("type %s not found in %s", id, path)
}

// gofmt pretty-prints e.
func (p Pkg) gofmt(e ast.Expr) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, p.FileSet, e)
	return buf.String()
}

// fullType returns the fully qualified type of e.
// Examples, assuming package net/http:
// 	fullType(int) => "int"
// 	fullType(Handler) => "http.Handler"
// 	fullType(io.Reader) => "io.Reader"
// 	fullType(*Request) => "*http.Request"
func (p Pkg) fullType(e ast.Expr) string {
	ast.Inspect(e, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			// Using typeSpec instead of IsExported here would be
			// more accurate, but it'd be crazy expensive, and if
			// the type isn't exported, there's no point trying
			// to implement it anyway.
			if n.IsExported() {
				n.Name = p.Package.Name + "." + n.Name
			}
		case *ast.SelectorExpr:
			return false
		}
		return true
	})
	return p.gofmt(e)
}

func (p Pkg) params(field *ast.Field) []Param {
	var params []Param
	typ := p.fullType(field.Type)
	for _, name := range field.Names {
		params = append(params, Param{Name: name.Name, Type: typ})
	}
	// Handle anonymous params
	if len(params) == 0 {
		params = []Param{Param{Type: typ}}
	}
	return params
}

// Method represents a method signature.
type Method struct {
	Recv            string
	RecVariableName string
	Func
}

// Func represents a function signature.
type Func struct {
	Name        string
	Params      []Param
	Res         []Param
	ReturnValue string
	Comments    string
}

// Param represents a parameter in a function or method signature.
type Param struct {
	Name string
	Type string
}

func (p Pkg) funcsig(f *ast.Field, cmap ast.CommentMap) Func {
	fn := Func{Name: f.Names[0].Name}
	typ := f.Type.(*ast.FuncType)
	if typ.Params != nil {
		for _, field := range typ.Params.List {
			for _, param := range p.params(field) {
				// only for method parameters:
				// assign a blank identifier "_" to an anonymous parameter
				if param.Name == "" {
					param.Name = "_"
				}
				fn.Params = append(fn.Params, param)
			}
		}
	}
	if typ.Results != nil {
		for _, field := range typ.Results.List {
			fn.Res = append(fn.Res, p.params(field)...)
		}
	}
	if commentsBefore(f, cmap.Comments()) {
		fn.Comments = flattenCommentMap(cmap)
	}
	return fn
}

// The error interface is built-in.
var errorInterface = []Func{{
	Name: "Error",
	Res:  []Param{{Type: "string"}},
}}

// funcs returns the set of methods required to implement iface.
// It is called funcs rather than methods because the
// function descriptions are functions; there is no receiver.
func funcs(iface string, srcDir string) ([]Func, error) {
	// Special case for the built-in error interface.
	if iface == "error" {
		return errorInterface, nil
	}

	// Locate the interface.
	path, id, err := findInterface(iface, srcDir)
	if err != nil {
		return nil, err
	}

	// Parse the package and find the interface declaration.
	p, spec, err := typeSpec(path, id, srcDir)
	if err != nil {
		return nil, fmt.Errorf("interface %s not found: %s", iface, err)
	}
	idecl, ok := spec.Type.(*ast.InterfaceType)
	if !ok {
		return nil, fmt.Errorf("not an interface: %s", iface)
	}

	if idecl.Methods == nil {
		return nil, fmt.Errorf("empty interface: %s", iface)
	}

	var fns []Func
	for _, fndecl := range idecl.Methods.List {
		if len(fndecl.Names) == 0 {
			// Embedded interface: recurse
			embedded, err := funcs(p.fullType(fndecl.Type), srcDir)
			if err != nil {
				return nil, err
			}
			fns = append(fns, embedded...)
			continue
		}

		fn := p.funcsig(fndecl, spec.CommentMap.Filter(fndecl))
		fns = append(fns, fn)
	}
	return fns, nil
}

const typeField = "{{.Name}}Func " +
	"func({{range .Params}}{{.Name}} {{.Type}}, {{end}})" +
	"({{range .Res}}{{.Name}} {{.Type}}, {{end}})\n"

var tmpl1 = template.Must(template.New("test").Parse(typeField))

// genTypeDefinition prints nicely formatted mock type fields
// If recv is not a valid receiver expression,
// genTypeDefinition will panic.
func genTypeDefinition(recv string, fns []Func) []byte {
	var buf bytes.Buffer
	mockTypeName := strings.Split(recv, " ")[1]
	mockTypeName = strings.TrimPrefix(mockTypeName, "*")
	buf.WriteString("type " + mockTypeName + " struct {\n")
	for _, fn := range fns {
		// skip function if it has an override return value
		if fn.ReturnValue != "" {
			continue
		}

		meth := Method{Func: fn}
		tmpl1.Execute(&buf, meth)
	}
	buf.WriteString("}\n\n")

	buf.WriteString(fmt.Sprintf("func New%s() *%s {\n  return &%s{}}\n\n", strings.Title(mockTypeName), mockTypeName, mockTypeName))

	return buf.Bytes()
}

const methodStub = "func ({{.Recv}}) {{.Name}}" +
	"({{range .Params}}{{.Name}} {{.Type}}, {{end}})" +
	"({{range .Res}}{{.Name}} {{.Type}}, {{end}})" +
	"{\n{{if .Res}}return {{end}}" + "{{.RecVariableName}}.{{.Name}}Func({{range .Params}}{{.Name}}, {{end}})" + "}\n\n"

var tmpl2 = template.Must(template.New("test").Parse(methodStub))

const methodStubOverride = "func ({{.Recv}}) {{.Name}}" +
	"({{range .Params}}{{.Name}} {{.Type}}, {{end}})" +
	"({{range .Res}}{{.Name}} {{.Type}}, {{end}})" +
	"{\n{{if .Res}}return {{end}}" + "{{.ReturnValue}}" + "}\n\n"

var tmpl2Override = template.Must(template.New("test").Parse(methodStubOverride))

// genMethodStubs prints nicely formatted method stubs
func genMethodStubs(recv string, fns []Func) []byte {
	var buf bytes.Buffer
	for _, fn := range fns {
		meth := Method{Recv: recv, RecVariableName: strings.Split(recv, " ")[0], Func: fn}
		if fn.ReturnValue != "" {
			tmpl2Override.Execute(&buf, meth)
		} else {
			tmpl2.Execute(&buf, meth)
		}
	}

	return buf.Bytes()
}

const stub = "{{if .Comments}}{{.Comments}}{{end}}" +
	"func ({{.Recv}}) {{.Name}}" +
	"({{range .Params}}{{.Name}} {{.Type}}, {{end}})" +
	"({{range .Res}}{{.Name}} {{.Type}}, {{end}})" +
	"{\n" + "panic(\"not implemented\") // TODO: Implement" + "\n}\n\n"

var tmpl = template.Must(template.New("test").Parse(stub))

// validReceiver reports whether recv is a valid receiver expression.
func validReceiver(recv string) bool {
	if recv == "" {
		// The parse will parse empty receivers, but we don't want to accept them,
		// since it won't generate a usable code snippet.
		return false
	}
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "", "package hack\nfunc ("+recv+") Foo()", 0)
	return err == nil
}

// commentsBefore reports whether commentGroups precedes a field.
func commentsBefore(field *ast.Field, cg []*ast.CommentGroup) bool {
	if len(cg) > 0 {
		return cg[0].Pos() < field.Pos()
	}
	return false
}

// flattenCommentMap flattens the comment map to a string.
// This function must be used at the point when m is expected to have a single
// element.
func flattenCommentMap(m ast.CommentMap) string {
	if len(m) != 1 {
		panic("flattenCommentMap expects comment map of length 1")
	}
	var result strings.Builder
	for _, cgs := range m {
		for _, cg := range cgs {
			for _, c := range cg.List {
				result.WriteString(c.Text)
				// add an end-of-line character if this is '//'-style comment
				if c.Text[1] == '/' {
					result.WriteString("\n")
				}
			}
		}
	}

	// for '/*'-style comments, make sure to append EOL character to the comment
	// block
	if s := result.String(); !strings.HasSuffix(s, "\n") {
		result.WriteString("\n")
	}

	return result.String()
}

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `
mockit generates a mock file that contains a mock which implements the given interface.

mockit [-dir directory] <recv> <iface> <packagename> [override-methods]

`[1:])
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, `

Examples:

mockit 'm *mockReader' io.Reader mocks > mocks/reader_mock_gen.go
override methods example: "InsideTx:f(ds)" "IsTx:true"

Don't forget the single quotes around the receiver type
to prevent shell globbing.
`[1:])
		os.Exit(2)
	}
	flag.Parse()

	if len(flag.Args()) < 3 {
		flag.Usage()
	}

	recv, iface, packageName := flag.Arg(0), flag.Arg(1), flag.Arg(2)
	if !validReceiver(recv) {
		fatal(fmt.Sprintf("invalid receiver: %q", recv))
	}

	argCount := flag.NArg()
	overrideCount := argCount - 3
	methodOverrides := make(map[string]string)
	for i := 0; i < overrideCount; i += 1 {
		rawOverride := flag.Arg(3 + i)
		overrideSlice := strings.SplitN(rawOverride, ":", 2)
		if len(overrideSlice) != 2 {
			fatal(fmt.Sprintf("Invalid method override: %s", rawOverride))
		}
		methodOverrides[overrideSlice[0]] = overrideSlice[1]
	}

	if *flagSrcDir == "" {
		if dir, err := os.Getwd(); err == nil {
			*flagSrcDir = dir
		}
	}

	src := fmt.Sprintf("// Code generated by mockit.\n// DO NOT EDIT!\n\npackage %s\n\n", packageName)

	fns, err := funcs(iface, *flagSrcDir)
	if err != nil {
		fatal(err)
	}

	for i := range fns {
		fns[i].ReturnValue = methodOverrides[fns[i].Name]
		delete(methodOverrides, fns[i].Name)
	}

	if len(methodOverrides) > 0 {
		fatal(fmt.Sprintf("Unused method overrides: %s", methodOverrides))
	}

	body := string(genTypeDefinition(recv, fns))
	body += string(genMethodStubs(recv, fns))

	// Remove package name prefix from types
	body = strings.Replace(body, packageName+".", "", -1)

	src += body

	pretty, err := format.Source([]byte(src))
	if err != nil {
		fmt.Println(src)
		fatal(err)
	}

	fmt.Print(string(pretty))
}

func fatal(msg interface{}) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

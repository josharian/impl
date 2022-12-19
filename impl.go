// impl generates method stubs for implementing an interface.
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
	flagSrcDir   = flag.String("dir", "", "package source directory, useful for vendored code")
	flagComments = flag.Bool("comments", true, "include interface comments in the generated stubs")
	flagRecvPkg  = flag.String("recvpkg", "", "package name of the receiver")
)

func parseTypeParams(in string) (string, []string, error) {
	firstOpenBracket := strings.Index(in, "[")
	if firstOpenBracket < 0 {
		return in, []string{}, nil
	}
	// there are type parameters in our interface
	id := in[:firstOpenBracket]
	firstCloseBracket := strings.LastIndex(in, "]")
	if firstCloseBracket < 0 {
		// make sure we're closing our list of type parameters
		return "", nil, fmt.Errorf("invalid interface name (cannot have [ without ]): %s", in)
	}
	if firstCloseBracket != len(in)-1 {
		// make sure the first close bracket is actually the last character of the interface name
		return "", nil, fmt.Errorf("invalid interface name (cannot have ] anywhere except the last character): %s", in)
	}
	params := strings.Split(in[firstOpenBracket+1:firstCloseBracket], ",")
	typeParams := make([]string, 0, len(params))
	for _, param := range params {
		typeParams = append(typeParams, strings.TrimSpace(param))
	}
	if len(typeParams) < 1 {
		// make sure if we're declaring type parameters, we declare at least one
		return "", nil, fmt.Errorf("invalid interface name (cannot have empty type parameters): %s", in)
	}
	return id, typeParams, nil
}

// findInterface returns the import path and identifier of an interface.
// For example, given "http.ResponseWriter", findInterface returns
// "net/http", "ResponseWriter".
// If a fully qualified interface is given, such as "net/http.ResponseWriter",
// it simply parses the input.
// If an unqualified interface such as "UserDefinedInterface" is given, then
// the interface definition is presumed to be in the package within srcDir and
// findInterface returns "", "UserDefinedInterface".
func findInterface(iface string, srcDir string) (path string, id string, typeParams []string, err error) {
	if len(strings.Fields(iface)) != 1 && !strings.Contains(iface, "[") {
		return "", "", nil, fmt.Errorf("couldn't parse interface: %s", iface)
	}

	srcPath := filepath.Join(srcDir, "__go_impl__.go")

	if slash := strings.LastIndex(iface, "/"); slash > -1 {
		// package path provided
		dot := strings.LastIndex(iface, ".")
		// make sure iface does not end with "/" (e.g. reject net/http/)
		if slash+1 == len(iface) {
			return "", "", nil, fmt.Errorf("interface name cannot end with a '/' character: %s", iface)
		}
		// make sure iface does not end with "." (e.g. reject net/http.)
		if dot+1 == len(iface) {
			return "", "", nil, fmt.Errorf("interface name cannot end with a '.' character: %s", iface)
		}
		// make sure iface has at least one "." after "/" (e.g. reject net/http/httputil)
		if strings.Count(iface[slash:], ".") == 0 {
			return "", "", nil, fmt.Errorf("invalid interface name: %s", iface)
		}
		path = iface[:dot]
		id = iface[dot+1:]
		id, typeParams, err = parseTypeParams(id)
		if err != nil {
			return "", "", nil, err
		}
		return path, id, typeParams, nil
	}

	src := []byte("package hack\n" + "var i " + iface)
	// If we couldn't determine the import path, goimports will
	// auto fix the import path.
	imp, err := imports.Process(srcPath, src, nil)
	if err != nil {
		return "", "", nil, fmt.Errorf("couldn't parse interface: %s", iface)
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
		return "", "", nil, fmt.Errorf("unrecognized interface: %s", iface)
	}

	if !qualified {
		// If !qualified, the code looks like:
		//
		// package hack
		//
		// var i Reader
		decl := f.Decls[0].(*ast.GenDecl)      // var i io.Reader
		spec := decl.Specs[0].(*ast.ValueSpec) // i io.Reader
		if indxExpr, ok := spec.Type.(*ast.IndexExpr); ok {
			// a generic type with one type parameter shows up as an IndexExpr
			id = indxExpr.X.(*ast.Ident).Name
			typeParams = append(typeParams, indxExpr.Index.(*ast.Ident).Name)
		} else if indxListExpr, ok := spec.Type.(*ast.IndexListExpr); ok {
			// a generic type with multiple type parameters shows up as an IndexListExpr
			id = indxListExpr.X.(*ast.Ident).Name
			for _, typeParam := range indxListExpr.Indices {
				typeParams = append(typeParams, typeParam.(*ast.Ident).Name)
			}
		} else {
			sel := spec.Type.(*ast.Ident)
			id = sel.Name // Reader
		}

		return path, id, typeParams, nil
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
	if indxExpr, ok := spec.Type.(*ast.IndexExpr); ok {
		// a generic type with one type parameter shows up as an IndexExpr
		id = indxExpr.X.(*ast.SelectorExpr).Sel.Name
		typeParams = append(typeParams, indxExpr.Index.(*ast.Ident).Name)
	} else if indxListExpr, ok := spec.Type.(*ast.IndexListExpr); ok {
		// a generic type with multiple type parameters shows up as an IndexListExpr
		id = indxListExpr.X.(*ast.SelectorExpr).Sel.Name
		for _, typeParam := range indxListExpr.Indices {
			typeParams = append(typeParams, typeParam.(*ast.Ident).Name)
		}
	} else {
		sel := spec.Type.(*ast.SelectorExpr) // io.Reader
		id = sel.Sel.Name                    // Reader
	}

	return path, id, typeParams, nil
}

// Pkg is a parsed build.Package.
type Pkg struct {
	*build.Package
	*token.FileSet
	// recvPkg is the package name of the function receiver
	recvPkg string
}

// Spec is ast.TypeSpec with the associated comment map.
type Spec struct {
	*ast.TypeSpec
	ast.CommentMap
	TypeParams map[string]string
}

// typeSpec locates the *ast.TypeSpec for type id in the import path.
func typeSpec(path, id string, typeParams []string, srcDir string) (Pkg, Spec, error) {
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

		for _, decl := range f.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			decl := genDecl
			if decl.Tok != token.TYPE {
				continue
			}
			for _, spec := range decl.Specs {
				spec := spec.(*ast.TypeSpec)
				if spec.Name.Name != id {
					continue
				}
				tParams := make(map[string]string, len(typeParams))
				if spec.TypeParams != nil {
					var specParamNames []string
					for _, typeParam := range spec.TypeParams.List {
						for _, name := range typeParam.Names {
							if name == nil {
								continue
							}
							specParamNames = append(specParamNames, name.Name)
						}
					}
					if len(specParamNames) != len(typeParams) {
						continue
					}
					for pos, specParamName := range specParamNames {
						tParams[specParamName] = typeParams[pos]
					}
				}
				p := Pkg{Package: pkg, FileSet: fset}
				s := Spec{TypeSpec: spec, TypeParams: tParams}
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
//
//	fullType(int) => "int"
//	fullType(Handler) => "http.Handler"
//	fullType(io.Reader) => "io.Reader"
//	fullType(*Request) => "*http.Request"
func (p Pkg) fullType(e ast.Expr) string {
	ast.Inspect(e, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			// Using typeSpec instead of IsExported here would be
			// more accurate, but it'd be crazy expensive, and if
			// the type isn't exported, there's no point trying
			// to implement it anyway.
			if n.IsExported() && p.recvPkg != p.Package.Name {
				n.Name = p.Package.Name + "." + n.Name
			}
		case *ast.SelectorExpr:
			return false
		}
		return true
	})
	return p.gofmt(e)
}

func (p Pkg) params(field *ast.Field, genericTypes map[string]string) []Param {
	var params []Param
	var typ string
	ident, ok := field.Type.(*ast.Ident)
	if !ok || ident == nil {
		typ = p.fullType(field.Type)
	} else if genType, ok := genericTypes[ident.Name]; ok {
		typ = genType
	} else {
		typ = p.fullType(field.Type)
	}
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
	Recv string
	Func
}

// Func represents a function signature.
type Func struct {
	Name     string
	Params   []Param
	Res      []Param
	Comments string
}

// Param represents a parameter in a function or method signature.
type Param struct {
	Name string
	Type string
}

// EmitComments specifies whether comments from the interface should be preserved in the implementation.
type EmitComments bool

const (
	WithComments    EmitComments = true
	WithoutComments EmitComments = false
)

func (p Pkg) funcsig(f *ast.Field, genericParams map[string]string, cmap ast.CommentMap, comments EmitComments) Func {
	fn := Func{Name: f.Names[0].Name}
	typ := f.Type.(*ast.FuncType)
	if typ.Params != nil {
		for _, field := range typ.Params.List {
			for _, param := range p.params(field, genericParams) {
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
			fn.Res = append(fn.Res, p.params(field, genericParams)...)
		}
	}
	if comments == WithComments && f.Doc != nil {
		fn.Comments = flattenDocComment(f)
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
func funcs(iface, srcDir, recvPkg string, comments EmitComments) ([]Func, error) {
	// Special case for the built-in error interface.
	if iface == "error" {
		return errorInterface, nil
	}

	// Locate the interface.
	path, id, typeParams, err := findInterface(iface, srcDir)
	if err != nil {
		return nil, err
	}

	// Parse the package and find the interface declaration.
	p, spec, err := typeSpec(path, id, typeParams, srcDir)
	if err != nil {
		return nil, fmt.Errorf("interface %s not found: %s", iface, err)
	}
	p.recvPkg = recvPkg

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
			embedded, err := funcs(p.fullType(fndecl.Type), srcDir, recvPkg, comments)
			if err != nil {
				return nil, err
			}
			fns = append(fns, embedded...)
			continue
		}

		fn := p.funcsig(fndecl, spec.TypeParams, spec.CommentMap.Filter(fndecl), comments)
		fns = append(fns, fn)
	}
	return fns, nil
}

const stub = "{{if .Comments}}{{.Comments}}{{end}}" +
	"func ({{.Recv}}) {{.Name}}" +
	"({{range .Params}}{{.Name}} {{.Type}}, {{end}})" +
	"({{range .Res}}{{.Name}} {{.Type}}, {{end}})" +
	"{\n" + "panic(\"not implemented\") // TODO: Implement" + "\n}\n\n"

var tmpl = template.Must(template.New("test").Parse(stub))

// genStubs prints nicely formatted method stubs
// for fns using receiver expression recv.
// If recv is not a valid receiver expression,
// genStubs will panic.
// genStubs won't generate stubs for
// already implemented methods of receiver.
func genStubs(recv string, fns []Func, implemented map[string]bool) []byte {
	var recvName string
	if recvs := strings.Fields(recv); len(recvs) > 1 {
		recvName = recvs[0]
	}

	// (r *recv) F(r string) {} => (r *recv) F(_ string)
	fixParams := func(params []Param) {
		for i, p := range params {
			if p.Name == recvName {
				params[i].Name = "_"
			}
		}
	}

	buf := new(bytes.Buffer)
	for _, fn := range fns {
		if implemented[fn.Name] {
			continue
		}

		fixParams(fn.Params)
		fixParams(fn.Res)
		meth := Method{Recv: recv, Func: fn}
		tmpl.Execute(buf, meth)
	}

	pretty, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}
	return pretty
}

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

// flattenDocComment flattens the field doc comments to a string
func flattenDocComment(f *ast.Field) string {
	var result strings.Builder
	for _, c := range f.Doc.List {
		result.WriteString(c.Text)
		// add an end-of-line character if this is '//'-style comment
		if c.Text[1] == '/' {
			result.WriteString("\n")
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
impl generates method stubs for recv to implement iface.

impl [-dir directory] <recv> <iface>

`[1:])
		flag.PrintDefaults()
		fmt.Fprint(os.Stderr, `

Examples:
		
impl 'f *File' io.Reader
impl Murmur hash.Hash
impl -dir $GOPATH/src/github.com/josharian/impl Murmur hash.Hash

Don't forget the single quotes around the receiver type
to prevent shell globbing.
`[1:])
		os.Exit(2)
	}
	flag.Parse()

	if len(flag.Args()) < 2 {
		flag.Usage()
	}

	recv, iface := flag.Arg(0), flag.Arg(1)
	if !validReceiver(recv) {
		fatal(fmt.Sprintf("invalid receiver: %q", recv))
	}

	if *flagSrcDir == "" {
		if dir, err := os.Getwd(); err == nil {
			*flagSrcDir = dir
		}
	}

	var recvPkg = *flagRecvPkg
	if recvPkg == "" {
		//  "   s *Struct   " , receiver: Struct
		recvs := strings.Fields(recv)
		receiver := recvs[len(recvs)-1] // note that this correctly handles "s *Struct" and "*Struct"
		receiver = strings.TrimPrefix(receiver, "*")
		pkg, _, err := typeSpec("", receiver, nil, *flagSrcDir)
		if err == nil {
			recvPkg = pkg.Package.Name
		}
	}

	fns, err := funcs(iface, *flagSrcDir, recvPkg, EmitComments(*flagComments))
	if err != nil {
		fatal(err)
	}

	// Get list of already implemented funcs
	implemented, err := implementedFuncs(fns, recv, *flagSrcDir)
	if err != nil {
		fatal(err)
	}

	src := genStubs(recv, fns, implemented)
	fmt.Print(string(src))
}

func fatal(msg interface{}) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

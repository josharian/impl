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

// Type is a parsed type reference.
type Type struct {
	// ID is the type's ID or name. For example, in "foo[Bar, Baz]", the ID
	// is "foo".
	ID string

	// Params are the type's type params. For example, in "foo[Bar, Baz]",
	// the Params are []string{"Bar", "Baz"}.
	//
	// Params never list the type of the "name type" construction of type
	// params used when defining a generic type. They will always be just
	// the filling type, as seen when using a generic type.
	//
	// Params will always be the type parameters only for the top-level
	// type; if the params themselves have type parameters, they will
	// remain joined to the type name. So "foo[Bar, Baz[Quux]]" will be
	// returned as {ID: "foo", Params: []string{"Bar", "Baz[Quux]"}}
	Params []string
}

// String constructs a reference to the Type. For example:
// Type{ID: "Foo", Params{"Bar", "Baz[[]Quux]"}}
// would yield
// Foo[Bar, Baz[[]Quux]]
func (t Type) String() string {
	var res strings.Builder
	res.WriteString(t.ID)
	if len(t.Params) < 1 {
		return res.String()
	}
	res.WriteString("[")
	res.WriteString(strings.Join(t.Params, ", "))
	res.WriteString("]")
	return res.String()
}

// parseType parses an interface reference into a Type, allowing us to
// distinguish between the interface's ID or name and its type parameters.
func parseType(in string) (Type, error) {
	expr, err := parser.ParseExpr(in)
	if err != nil {
		return Type{}, err
	}
	return typeFromAST(expr)
}

// findInterface returns the import path and type of an interface.
// For example, given "http.ResponseWriter", findInterface returns
// "net/http", Type{ID: "ResponseWriter"}.
// If a fully qualified interface is given, such as "net/http.ResponseWriter",
// it simply parses the input.
// If an unqualified interface such as "UserDefinedInterface" is given, then
// the interface definition is presumed to be in the package within srcDir and
// findInterface returns "", Type{ID: "UserDefinedInterface"}.
//
// Generic types will have their type params returned in the Params property of
// the Type. Input should always reference generic types with their parameters
// filled, i.e. GenericType[string, bool], as opposed to
// GenericType[A any, B comparable].
func findInterface(input string, srcDir string) (path string, iface Type, err error) {
	if len(strings.Fields(input)) != 1 && !strings.Contains(input, "[") {
		return "", Type{}, fmt.Errorf("couldn't parse interface: %s", input)
	}

	srcPath := filepath.Join(srcDir, "__go_impl__.go")

	if slash := strings.LastIndex(input, "/"); slash > -1 {
		// package path provided
		dot := strings.LastIndex(input, ".")
		// make sure iface does not end with "/" (e.g. reject net/http/)
		if slash+1 == len(input) {
			return "", Type{}, fmt.Errorf("interface name cannot end with a '/' character: %s", input)
		}
		// make sure iface does not end with "." (e.g. reject net/http.)
		if dot+1 == len(input) {
			return "", Type{}, fmt.Errorf("interface name cannot end with a '.' character: %s", input)
		}
		// make sure iface has at least one "." after "/" (e.g. reject net/http/httputil)
		if strings.Count(input[slash:], ".") == 0 {
			return "", Type{}, fmt.Errorf("invalid interface name: %s", input)
		}
		path = input[:dot]
		id := input[dot+1:]
		iface, err = parseType(id)
		if err != nil {
			return "", Type{}, err
		}
		return path, iface, nil
	}

	src := []byte("package hack\n" + "var i " + input)
	// If we couldn't determine the import path, goimports will
	// auto fix the import path.
	imp, err := imports.Process(srcPath, src, nil)
	if err != nil {
		return "", Type{}, fmt.Errorf("couldn't parse interface: %s", input)
	}

	// imp should now contain an appropriate import.
	// Parse out the import and the identifier.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, srcPath, imp, 0)
	if err != nil {
		panic(err)
	}

	qualified := strings.Contains(input, ".")

	if len(f.Imports) == 0 && qualified {
		return "", Type{}, fmt.Errorf("unrecognized interface: %s", input)
	}

	if !qualified {
		// If !qualified, the code looks like:
		//
		// package hack
		//
		// var i Reader
		decl := f.Decls[0].(*ast.GenDecl)      // var i io.Reader
		spec := decl.Specs[0].(*ast.ValueSpec) // i io.Reader
		iface, err = typeFromAST(spec.Type)
		return path, iface, err
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
	iface, err = typeFromAST(spec.Type)
	return path, iface, err
}

func typeFromAST(in ast.Expr) (Type, error) {
	switch specType := in.(type) {
	case *ast.Ident:
		// a standalone identifier (Reader) shows up as an Ident
		return Type{ID: specType.Name}, nil
	case *ast.SelectorExpr:
		// an identifier in a different package (io.Reader) shows up as a SelectorExpr
		// we need to pull the name out
		return Type{ID: specType.Sel.Name}, nil
	case *ast.StarExpr:
		// pointer identifiers (*Reader) show up as a StarExpr
		// we need to pull the name out and prefix it with a *
		typ, err := typeFromAST(specType.X)
		if err != nil {
			return Type{}, err
		}
		typ.ID = "*" + typ.ID
		return typ, nil
	case *ast.ArrayType:
		// slices and arrays ([]Reader) show up as an ArrayType
		typ, err := typeFromAST(specType.Elt)
		if err != nil {
			return Type{}, err
		}
		prefix := "["
		if specType.Len != nil {
			prefix += specType.Len.(*ast.BasicLit).Value
		}
		prefix += "]"
		typ.ID = prefix + typ.ID
		return typ, nil
	case *ast.MapType:
		// maps (map[string]Reader) show up as a MapType
		key, err := typeFromAST(specType.Key)
		if err != nil {
			return Type{}, err
		}
		value, err := typeFromAST(specType.Value)
		if err != nil {
			return Type{}, err
		}
		return Type{
			ID: "map[" + key.String() + "]" + value.String(),
		}, nil
	case *ast.FuncType:
		// funcs (func() Reader) show up as a FuncType
		// NOTE: we don't actually parse out the type params of a FuncType.
		// This should be okay, because we really only care about
		// parsing out the type params when parsing interface
		// identifiers. And FuncTypes never signify an interface
		// identifier, they're just an argument to it or a type param
		// of it.
		// We don't parse them out anyways like we do for everything
		// else because funcs, alone, are pretty weird in how they use
		// generics.
		// For everything else, it's identifier[params].
		// For funcs, the params get stuck in the middle of the identifier:
		// func Foo[Param1, Param2](context.Context, Param1) Param2
		// We're gonna need to complicate everything to support that
		// construction, and we don't actually need the deconstructed
		// bits, so we're just... not going to deconstruct it at all.
		var res strings.Builder
		res.WriteString("func")
		if specType.TypeParams != nil && len(specType.TypeParams.List) > 0 {
			res.WriteString("[")
			paramList, err := buildFuncParamList(specType.TypeParams.List)
			if err != nil {
				return Type{}, err
			}
			res.WriteString(paramList)
			res.WriteString("]")
		}
		res.WriteString("(")
		if specType.Params != nil {
			paramList, err := buildFuncParamList(specType.Params.List)
			if err != nil {
				return Type{}, err
			}
			res.WriteString(paramList)
		}
		res.WriteString(")")
		if specType.Results != nil && len(specType.Results.List) > 0 {
			res.WriteString(" ")
			if len(specType.Results.List) > 1 {
				res.WriteString("(")
			}
			paramList, err := buildFuncParamList(specType.Results.List)
			if err != nil {
				return Type{}, err
			}
			res.WriteString(paramList)
			if len(specType.Results.List) > 1 {
				res.WriteString(")")
			}
		}
		return Type{ID: res.String()}, nil
	case *ast.ChanType:
		var res strings.Builder
		// channels (chan Reader) show up as a ChanType
		// we need to be careful to preserve send/receive semantics
		if specType.Dir&ast.SEND == 0 {
			// this is a receive-only channel, write the arrow before the chan keyword
			res.WriteString("<-")
		}
		res.WriteString("chan")
		if specType.Dir&ast.RECV == 0 {
			// this is a send-only channel, write the arrow after the chan keyword
			res.WriteString("<-")
		}
		res.WriteString(" ")
		valType, err := typeFromAST(specType.Value)
		if err != nil {
			return Type{}, err
		}
		valType.ID = res.String() + valType.ID
		return valType, nil
	case *ast.IndexExpr:
		// a generic type with one type parameter (Reader[Foo]) shows up as an IndexExpr
		id, err := typeFromAST(specType.X)
		if err != nil {
			return Type{}, err
		}
		if len(id.Params) > 0 {
			return Type{}, fmt.Errorf("got type parameters for a type ID, which is very confusing: %s", id.String())
		}
		param, err := typeFromAST(specType.Index)
		if err != nil {
			return Type{}, err
		}
		return Type{
			ID:     id.ID,
			Params: []string{param.String()},
		}, nil
	case *ast.IndexListExpr:
		// a generic type with multiple type parameters shows up as an IndexListExpr
		id, err := typeFromAST(specType.X)
		if err != nil {
			return Type{}, err
		}
		if len(id.Params) > 0 {
			return Type{}, fmt.Errorf("got type parameters for a type ID, which is very confusing: %s", id.String())
		}
		res := Type{
			ID: specType.X.(*ast.Ident).Name,
		}
		for _, typeParam := range specType.Indices {
			param, err := typeFromAST(typeParam)
			if err != nil {
				return Type{}, err
			}
			res.Params = append(res.Params, param.String())
		}
		return res, nil
	}
	return Type{}, fmt.Errorf("unexpected AST type %T", in)
}

// buildFuncParamList returns a string representation of a list of function
// params (type params, function arguments, returns) given an []*ast.Field
// for those things.
func buildFuncParamList(list []*ast.Field) (string, error) {
	var res strings.Builder
	for pos, field := range list {
		for namePos, name := range field.Names {
			res.WriteString(name.Name)
			if namePos+1 < len(field.Names) {
				res.WriteString(", ")
			}
		}
		if len(field.Names) > 0 {
			res.WriteString(" ")
		}
		fieldType, err := typeFromAST(field.Type)
		if err != nil {
			return "", err
		}
		res.WriteString(fieldType.String())
		if pos+1 < len(list) {
			res.WriteString(", ")
		}
	}
	return res.String(), nil
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
func typeSpec(path string, typ Type, srcDir string) (Pkg, Spec, error) {
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
			decl, ok := decl.(*ast.GenDecl)
			if !ok || decl.Tok != token.TYPE {
				continue
			}
			for _, spec := range decl.Specs {
				spec := spec.(*ast.TypeSpec)
				if spec.Name.Name != typ.ID {
					continue
				}
				typeParams, ok := matchTypeParams(spec, typ.Params)
				if !ok {
					continue
				}
				p := Pkg{Package: pkg, FileSet: fset}
				s := Spec{TypeSpec: spec, TypeParams: typeParams}
				return p, s, nil
			}
		}
	}
	return Pkg{}, Spec{}, fmt.Errorf("type %s not found in %s", typ.ID, path)
}

// matchTypeParams returns a map of type parameters from a parsed interface
// definition and the types that fill them from the user's specified type
// info. If the passed params can't be used to fill the type parameters on the
// passed type, a nil map and false are returned. No type checking is done,
// only that there are sufficient types to match.
func matchTypeParams(spec *ast.TypeSpec, params []string) (map[string]string, bool) {
	res := make(map[string]string, len(params))
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
		if len(specParamNames) != len(params) {
			return nil, false
		}
		for pos, specParamName := range specParamNames {
			res[specParamName] = params[pos]
		}
	}
	return res, true
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

func (p Pkg) params(field *ast.Field, typeParams map[string]string) []Param {
	var params []Param
	var typ string
	switch expr := field.Type.(type) {
	case *ast.Ident:
		if genType, ok := typeParams[expr.Name]; ok {
			typ = genType
		} else {
			typ = p.fullType(field.Type)
		}
	default:
		typ = p.fullType(field.Type)
	}
	for _, name := range field.Names {
		params = append(params, Param{Name: name.Name, Type: typ})
	}
	// Handle anonymous params
	if len(params) == 0 {
		params = []Param{{Type: typ}}
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

func (p Pkg) funcsig(f *ast.Field, typeParams map[string]string, cmap ast.CommentMap, comments EmitComments) Func {
	fn := Func{Name: f.Names[0].Name}
	typ := f.Type.(*ast.FuncType)
	if typ.Params != nil {
		for _, field := range typ.Params.List {
			for _, param := range p.params(field, typeParams) {
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
			fn.Res = append(fn.Res, p.params(field, typeParams)...)
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
	path, typ, err := findInterface(iface, srcDir)
	if err != nil {
		return nil, err
	}

	// Parse the package and find the interface declaration.
	p, spec, err := typeSpec(path, typ, srcDir)
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
		pkg, _, err := typeSpec("", Type{ID: receiver}, *flagSrcDir)
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

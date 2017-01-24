package impl

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
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

// An Implementer can, for a certain directory, create and/or update
// implementation with Go source code for a particular interface
type Implementer struct {
	Recv, IFace, Dir string

	funcs []Func

	recvName string
	typeDecl *ast.GenDecl
	methods  map[string]*ast.FuncDecl

	found bool

	file map[string]*ast.Package
	fset *token.FileSet
	buf  *bytes.Buffer
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

// Position returns, if found, the token.Position of the end of the type
// declaration for the specified receiver.
func (i *Implementer) Position() (*token.Position, error) {
	err := i.init()
	if err != nil {
		return nil, err
	}

	p := i.fset.Position(i.typeDecl.End())

	return &p, nil
}

// GenStubs prints nicely formatted method stubs for fns using receiver
// expression recv. If the Implementer is not in a valid state, or an error
// occurs, the error will be returned.
func (i *Implementer) GenStubs() ([]byte, error) {
	err := i.init()
	if err != nil {
		return nil, err
	}

	for _, fn := range i.funcs {
		if _, ok := i.methods[fn.Name]; !ok {
			meth := Method{Recv: i.Recv, Func: fn}
			tmpl.Execute(i.buf, meth)
		}
	}

	return format.Source(i.buf.Bytes())
}

// ensureOffset will ensure that, given a file:line:col generated position, the
// offset is correct for the file.
func ensureOffset(p *token.Position) error {
	if p.Offset != 0 || (p.Line == 0 && p.Column == 0) {
		return nil
	}

	bs, err := ioutil.ReadFile(p.Filename)
	if err != nil {
		return err
	}

	col, line := 1, 1

	for i := range bs {
		col++
		if line == p.Line && col == p.Column {
			p.Offset = i
			return nil
		}
		if bs[i] == '\n' {
			col = 0
			line++
			continue
		}
	}
	return fmt.Errorf("Could not find %s", p)
}

// GenForPosition allows users to have more flexible stub generation, with the
// ability to specify exactly where the implementation should be generated. If
// the token.Position argument is nil, the generated code will be inserted
// immediately after the receiving type's declaration.
func (i *Implementer) GenForPosition(p *token.Position) ([]byte, error) {
	src, err := i.GenStubs()
	if err != nil {
		return nil, err
	}

	newline := []byte("\n\n")

	src = bytes.Join([][]byte{newline, src, newline}, nil)

	i.walk()

	if !i.found {
		return nil, fmt.Errorf("requested receiver not found: %s", i.recvName)
	}

	if p == nil {
		pp := i.fset.Position(i.typeDecl.End())
		p = &pp
	}

	err = ensureOffset(p)
	if err != nil {
		return nil, err
	}

	orig, err := ioutil.ReadFile(p.Filename)
	if err != nil {
		return nil, err
	}

	result := &bytes.Buffer{}

	result.Write(orig[:p.Offset])
	result.Write(src)
	result.Write(orig[p.Offset:])

	return format.Source(result.Bytes())
}

// validReceiver reports whether recv is a valid receiver expression.
func (i *Implementer) validateReceiver() error {
	err := i.init()
	if err != nil {
		return err
	}

	if i.Recv == "" {
		// The parse will parse empty receivers, but we don't want to accept them,
		// since it won't generate a usable code snippet.
		return fmt.Errorf("receiver was the empty string")
	}
	return nil
}

func (i *Implementer) init() error {
	if i.buf != nil {
		// Already initialized
		return nil
	}
	i.buf = &bytes.Buffer{}
	i.file = map[string]*ast.Package{}
	i.methods = map[string]*ast.FuncDecl{}
	if i.Recv == "" || i.IFace == "" {
		return fmt.Errorf("Receiver and interface must both be specified")
	}

	if i.Dir == "" {
		i.Dir = "."
	}

	err := i.validateReceiver()
	if err != nil {
		return err
	}

	i.fset = token.NewFileSet()
	i.file, err = parser.ParseDir(i.fset, i.Dir, nil, 0)

	i.funcs, err = funcs(i.IFace)
	if err != nil {
		return err
	}

	return i.walk()
}

func (i *Implementer) walk() error {
	if i.found {
		return nil
	}

	var err error

	i.recvName, err = getType(i.Recv)
	if err != nil {
		return err
	}

	for _, pkg := range i.file {
		for _, file := range pkg.Files {
			if !i.found {
				gen, _ := findTopTypeDecl(i.recvName, file)
				if gen != nil {
					i.found = true
					i.typeDecl = gen
				}
			}

			for _, meth := range getMethods(i.IFace, file) {
				i.methods[meth.Name.Name] = meth
			}
		}
	}

	return nil
}

package main

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

func (i *Implementer) Visit(node ast.Node) (w ast.Visitor) {
	if node == nil {
		return nil
	}

	switch node := node.(type) {
	case *ast.GenDecl:
		// Replace the type declaration reference until the top-level type
		// declaration with matching type name is found.
		if !i.found && node.Tok == token.TYPE {
			i.typeDecl = node
			return i
		}
	case *ast.TypeSpec:
		if getIdent(node, 0) == i.recvName {
			i.found = true
		}
	case *ast.FuncDecl:
		if node.Recv != nil && node.Name != nil {
			for _, r := range node.Recv.List {
				name := getIdent(r.Type, -1)
				if name == i.recvName {
					i.methods[node.Name.Name] = node
				}
			}
		}
	case *ast.File:
		//Continue parsing files
		return i
	}

	return nil
}

func (i *Implementer) Position() (*token.Position, error) {
	err := i.init()
	if err != nil {
		return nil, err
	}

	p := i.fset.Position(i.typeDecl.End())

	return &p, nil
}

// genStubs prints nicely formatted method stubs
// for fns using receiver expression recv.
// If recv is not a valid receiver expression,
// genStubs will panic.
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

func (i *Implementer) GenForPosition(p *token.Position) ([]byte, error) {
	src, err := i.GenStubs()
	if err != nil {
		return nil, err
	}

	newline := []byte("\n\n")

	src = bytes.Join([][]byte{newline, src, newline}, nil)

	if !i.found {
		return nil, fmt.Errorf("requested receiver not found")
	}

	if p == nil {
		pp := i.fset.Position(i.typeDecl.End())
		p = &pp
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
	i.fset = token.NewFileSet()

	i.file, err = parser.ParseDir(i.fset, i.Dir, nil, 0)

	return err
}

func (i *Implementer) init() error {
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

	i.funcs, err = funcs(i.IFace)
	if err != nil {
		return err
	}

	return i.walk()
}

func (i *Implementer) walk() error {
	var err error

	i.recvName, err = getType(i.Recv)
	if err != nil {
		return err
	}

	for _, pkg := range i.file {
		for _, file := range pkg.Files {
			ast.Walk(i, file)
		}
	}

	return nil
}

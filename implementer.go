package main

//TODO refactor to pkg/impl package

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"reflect"
	"strings"
)

// visitorFunc is a very simplistic implementation of an ast.Visitor based on a
// function that returns true if visiting (ast.Walk) should continue.
type visitorFunc func(ast.Node) (proceed bool)

func (v visitorFunc) Visit(node ast.Node) (w ast.Visitor) {
	if v(node) {
		return v
	}
	return nil
}

func getLastIdent(node ast.Node) string {
	var ident string

	ast.Inspect(node, func(child ast.Node) bool {
		if child, ok := child.(*ast.Ident); ok {
			ident = child.Name
		}
		return true
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

	return getLastIdent(a), nil
}

type implementer struct {
	recv, iface string
	funcs       []Func

	recvName string
	typeSpec *ast.TypeSpec
	ident    *ast.Ident
	methods  map[string]*ast.FuncDecl

	file map[string]*ast.Package
	fset *token.FileSet
	buf  *bytes.Buffer
}

func (i *implementer) Visit(node ast.Node) (w ast.Visitor) {
	if node == nil {
		return nil
	}

	log.Println(node, reflect.ValueOf(node).Type())

	switch n := node.(type) {
	case *ast.TypeSpec:
		// If we haven't found a matching top-level identifier yet, keep storing the most
		// recent TypeSpec
		if i.ident == nil {

			// If we find a typeSpec inside our current top-level typespec, ignore it
			if i.typeSpec != nil && n.Pos() < i.typeSpec.End() {
				log.Printf("nother typespec; new: %s, existing: %s\n", n.Name.Name, i.typeSpec.Name.Name)
				return nil
			}

			i.typeSpec = n
		}
		return i
	case *ast.Ident:
		// Once we find an identifier whose name matches
		if i.ident == nil {
			if n.Name != i.recvName {
				// If the name does not match (therefore this is not the typespec's
				// first identifier and thus the name of a type, stop processing this
				// portion of the tree.
				return nil
			}
			i.ident = n
		}
	case *ast.FuncDecl:
		if n.Recv != nil && n.Name != nil {
			for _, r := range n.Recv.List {
				typeName := getLastIdent(r.Type)
				if typeName == i.recvName {
					i.methods[n.Name.Name] = n
				}
			}
		}

		return nil
	// Only parse top-level identifiers
	case *ast.File:
		return i
	}
	return nil
}

// genStubs prints nicely formatted method stubs
// for fns using receiver expression recv.
// If recv is not a valid receiver expression,
// genStubs will panic.
func (i *implementer) genStubs() ([]byte, error) {
	for _, fn := range i.funcs {
		meth := Method{Recv: i.recv, Func: fn}
		tmpl.Execute(i.buf, meth)
	}

	var err error
	i.recvName, err = getType(i.recv)
	if err != nil {
		return nil, err
	}

	for _, pkg := range i.file {
		for _, file := range pkg.Files {

			ast.Walk(i, file)

			if i.typeSpec != nil {
				log.Println(i.fset.Position(i.typeSpec.End()))
			}
		}
	}

	return format.Source(i.buf.Bytes())
}

// validReceiver reports whether recv is a valid receiver expression.
func (i *implementer) validateReceiver() error {
	if i.recv == "" {
		// The parse will parse empty receivers, but we don't want to accept them,
		// since it won't generate a usable code snippet.
		return fmt.Errorf("receiver was the empty string")
	}
	i.fset = token.NewFileSet()

	var err error
	i.file, err = parser.ParseDir(i.fset, ".", nil, 0)

	return err
}

func (i *implementer) init(args []string) error {
	i.buf = &bytes.Buffer{}
	i.file = map[string]*ast.Package{}
	i.methods = map[string]*ast.FuncDecl{}
	if len(args) != 3 {
		return fmt.Errorf("Wrong number of arguments. Expected 2, got [\"%s\"]", strings.Join(args, "\", \""))
	}

	i.recv, i.iface = args[1], args[2]

	err := i.validateReceiver()

	if err != nil {
		return err
	}

	i.funcs, err = funcs(i.iface)
	if err != nil {
		return err
	}

	return nil
}

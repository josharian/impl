package impl

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/buildutil"
)

// An Implementer can, for a certain directory, create and/or update
// implementation with Go source code for a particular interface
type Implementer struct {
	Recv, IFace, Dir string

	Ctxt  *build.Context
	Input io.Reader

	funcs []Func

	recvName string
	typeDecl *ast.GenDecl
	methods  map[string]*ast.FuncDecl

	found bool

	file map[string]*ast.File
	fset *token.FileSet
	buf  *bytes.Buffer
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
		return nil, fmt.Errorf("error initializing implementer: %s", err)
	}

	for _, fn := range i.funcs {
		if _, ok := i.methods[fn.Name]; !ok {
			meth := Method{Recv: i.Recv, Func: fn}
			tmpl.Execute(i.buf, meth)
		}
	}

	bs, err := format.Source(i.buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("error formatting source: %s", err)
	}

	return bs, nil
}

// ensureOffset will ensure that, given a file:line:col generated position, the
// offset is correct for the file.
func (i *Implementer) ensureOffset(p *token.Position) error {
	if p.Offset != 0 || (p.Line == 0 && p.Column == 0) {
		return nil
	}

	f, err := i.Ctxt.OpenFile(p.Filename)
	if err != nil {
		return err
	}
	defer f.Close()

	bs, err := ioutil.ReadAll(f)
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

// getPositions takes a position identifier (file:line:char) and returns a
// golang tokenizer position
func (i *Implementer) getPosition(pos string) (*token.Position, error) {
	arr := strings.Split(pos, ":")

	if len(arr) < 2 {
		return nil, fmt.Errorf("Invalid position spec")
	}

	p := token.Position{Column: 1}

	p.Filename = arr[0]

	line, err := strconv.Atoi(arr[1])
	if err != nil {
		return nil, fmt.Errorf("invalid line spec in position: %s", err)
	}
	p.Line = line

	if len(arr) == 3 {
		col, err := strconv.Atoi(arr[2])
		if err != nil {
			return nil, fmt.Errorf("invalid column spec in position: %s", err)
		}
		p.Column = col
	}

	return &p, nil
}

// GenForPosition allows users to have more flexible stub generation, with the
// ability to specify exactly where the implementation should be generated. If
// the token.Position argument is nil, the generated code will be inserted
// immediately after the receiving type's declaration.
func (i *Implementer) GenForPosition(pos string) ([]byte, error) {
	i.init()

	src, err := i.GenStubs()
	if err != nil {
		return nil, err
	}

	p, err := i.getPosition(pos)
	if err != nil {
		return nil, err
	}

	newline := []byte("\n\n")

	src = bytes.Join([][]byte{newline, src, newline}, nil)

	if !i.found {
		return nil, fmt.Errorf("requested receiver not found: %s", i.recvName)
	}

	if p == nil {
		pp := i.fset.Position(i.typeDecl.End())
		p = &pp
	}

	err = i.ensureOffset(p)
	if err != nil {
		return nil, err
	}

	f, err := i.Ctxt.OpenFile(p.Filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	orig, err := ioutil.ReadAll(f)
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

func (i *Implementer) initContext() error {
	if i.Ctxt == nil {
		i.Ctxt = &build.Default
	}

	if i.Input != nil {
		modified, err := buildutil.ParseOverlayArchive(i.Input)
		if err != nil {
			return err
		}

		i.Ctxt = buildutil.OverlayContext(i.Ctxt, modified)
	}

	return nil
}

func (i *Implementer) init() error {
	if i.buf != nil {
		// Already initialized
		return nil
	}

	err := i.initContext()
	if err != nil {
		return err
	}

	i.buf = &bytes.Buffer{}
	i.methods = map[string]*ast.FuncDecl{}
	if i.Recv == "" || i.IFace == "" {
		return fmt.Errorf("Receiver and interface must both be specified")
	}

	err = i.validateReceiver()
	if err != nil {
		return err
	}

	if i.Dir == "" || i.Dir == "." {
		d, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return err
		}
		i.Dir = d
	}

	pkg, err := i.Ctxt.ImportDir(i.Dir, 0)
	if err != nil {
		return fmt.Errorf("Implementer.init() error importing directory %q: %s", i.Dir, err)
	}

	i.fset = token.NewFileSet()
	i.file = map[string]*ast.File{}

	for _, fname := range pkg.GoFiles {
		file, err := i.Ctxt.OpenFile(path.Join(i.Dir, fname))

		if err != nil {
			return err
		}
		defer file.Close()

		astFile, err := parser.ParseFile(i.fset, fname, file, 0)
		if err != nil {
			return err
		}
		i.file[fname] = astFile
	}

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

	for _, file := range i.file {
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

	return nil
}

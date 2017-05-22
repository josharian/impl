package impl

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getTestFile() (*token.FileSet, *ast.File) {
	fs := token.NewFileSet()

	file, _ := parser.ParseFile(fs, "./test/testmain.go", nil, 0)

	return fs, file
}

func TestGetIdent(t *testing.T) {
	asrt := assert.New(t)

	_, file := getTestFile()

	asrt.Equal("tester", getIdent(file, 0))

}

func TestGetMethods(t *testing.T) {
	asrt := assert.New(t)

	_, file := getTestFile()

	methods := getMethods("c", file)

	asrt.Len(methods, 1)
	asrt.Equal("Read", methods[0].Name.Name)
}

func formatArchive(m map[string]string) io.Reader {
	buf := &bytes.Buffer{}

	for name, contents := range m {
		fmt.Fprintln(buf, name)
		fmt.Fprintln(buf, float64(len(contents)))
		fmt.Fprint(buf, contents)
	}

	return buf
}

func TestOverlay(t *testing.T) {
	asrt := assert.New(t)

	file := formatArchive(map[string]string{
		"./test.go": `package tester
type aa struct {}`})

	i := Implementer{
		Archive: file,
		IFace:   "io.Reader",
		Recv:    "aa",
		Dir:     "./test",
	}

	bs, err := i.GenStubs()
	asrt.NoError(err)

	asrt.Equal(string(bs), `func (aa) Read(p []byte) (n int, err error) {
	panic("not implemented")
}

`)
}

type testPos interface {
	Test() string
}

func TestPosition(t *testing.T) {
	asrt := assert.New(t)

	file := formatArchive(map[string]string{
		"test.go": `package tester
type aa struct {}`})

	i := Implementer{
		Archive: file,
		IFace:   "testPos",
		Recv:    "aa",
		Dir:     "test",
	}

	bs, err := i.GenForPosition("test.go:2")
	asrt.NoError(err)
	log.Println(string(bs))
}

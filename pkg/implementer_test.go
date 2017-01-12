package impl

import (
	"go/ast"
	"go/parser"
	"go/token"
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

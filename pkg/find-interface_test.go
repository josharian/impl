package impl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindTopLevel(t *testing.T) {
	asrt := assert.New(t)

	_, file := getTestFile()
	a, b := findTopTypeDecl("a", file)

	asrt.NotNil(a)
	asrt.NotNil(b)

	asrt.Equal("a", b.Name.Name)
}

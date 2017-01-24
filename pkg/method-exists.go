package impl

import (
	"fmt"
	"go/ast"
	"strings"
)

// TODO implement impl.Func == ast.FuncDecl checking

// ErrMethodExists will be returned when a method should be created but an
// existing method already exists for the given receiver.
type ErrMethodExists struct {
	Wanted Func
	Exists ast.FuncDecl
}

func (e *ErrMethodExists) Error() string {
	args := []string{}
	for _, p := range e.Wanted.Params {
		args = append(args, p.Name+" "+p.Type)
	}
	ret := []string{}
	for _, r := range e.Wanted.Res {
		ret = append(ret, r.Name+" "+r.Type)
	}

	sig := fmt.Sprintf("%s(%s) (%s)", e.Wanted.Name, strings.Join(args, ", "), strings.Join(ret, ", "))

	return fmt.Sprintf("wanted to create Method %q, but this method name already exists for the receiver", sig)
}

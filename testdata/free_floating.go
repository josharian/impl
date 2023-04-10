package testdata

// Interface9Output is the expected output generated from reflecting on
// Interface9, provided that the receiver is equal to 'r *Receiver'.
var Interface9Output = `// Method1 is the first method of Interface1.
// line two
func (r *Receiver) Method1(arg1 string, arg2 string) (result string, err error) {
	panic("not implemented") // TODO: Implement
}

`

// Interface9 is a dummy interface to test the program output.
// This interface tests free-floating comments
type Interface9 interface {
	// free-floating comment before Method1

	// Method1 is the first method of Interface1.
	// line two
	Method1(arg1 string, arg2 string) (result string, err error)

	// free-floating comment after Method1
}

// free-floating comment at end of file. This must be the last comment in this file.

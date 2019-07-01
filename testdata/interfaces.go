package testdata

// Interface1 is a dummy interface to test the program output.
// This interface tests //-style method comments.
type Interface1 interface {
	// Method1 is the first method of Interface1.
	Method1(arg1 string, arg2 string) (result string, err error)
	// Method2 is the second method of Interface1.
	Method2(arg1 int, arg2 int) (result int, err error)
	// Method3 is the third method of Interface1.
	Method3(arg1 bool, arg2 bool) (result bool, err error)
}

// Interface2 is a dummy interface to test the program output.
// This interface tests /*-style method comments.
type Interface2 interface {
	/*
		Method1 is the first method of Interface2.
	*/
	Method1(arg1 int64, arg2 int64) (result int64, err error)
	/*
		Method2 is the second method of Interface2.
	*/
	Method2(arg1 float64, arg2 float64) (result float64, err error)
	/*
		Method3 is the third method of Interface2.
	*/
	Method3(arg1 interface{}, arg2 interface{}) (result interface{}, err error)
}

// Interface1Output is the expected output generated from reflecting on
// Interface1, provided that the receiver is equal to 'r *Receiver'.
var Interface1Output = `// Method1 is the first method of Interface1.
func (r *Receiver) Method1(arg1 string, arg2 string) (result string, err error) {
	panic("not implemented")
}

// Method2 is the second method of Interface1.
func (r *Receiver) Method2(arg1 int, arg2 int) (result int, err error) {
	panic("not implemented")
}

// Method3 is the third method of Interface1.
func (r *Receiver) Method3(arg1 bool, arg2 bool) (result bool, err error) {
	panic("not implemented")
}

`

// Interface2Output is the expected output generated from reflecting on
// Interface2, provided that the receiver is equal to 'r *Receiver'.
var Interface2Output = `/*
	Method1 is the first method of Interface2.
*/
func (r *Receiver) Method1(arg1 int64, arg2 int64) (result int64, err error) {
	panic("not implemented")
}

/*
	Method2 is the second method of Interface2.
*/
func (r *Receiver) Method2(arg1 float64, arg2 float64) (result float64, err error) {
	panic("not implemented")
}

/*
	Method3 is the third method of Interface2.
*/
func (r *Receiver) Method3(arg1 interface{}, arg2 interface{}) (result interface{}, err error) {
	panic("not implemented")
}

`

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

// Interface3 is a dummy interface to test the program output. This interface
// tests generation of method parameters and results.
//
// The first method tests the generation of anonymous method paramaters and
// results.
//
// The second method tests the generation of method parameters and results where
// the blank identifier "_" is already used in the names.
//
// The third method tests the generation of method parameters and results that
// are grouped by type.
type Interface3 interface {
	// Method1 is the first method of Interface3.
	Method1(string, string) (string, error)
	// Method2 is the second method of Interface3.
	Method2(_ int, arg2 int) (_ int, err error)
	// Method3 is the third method of Interface3.
	Method3(arg1, arg2 bool) (result1, result2 bool)
}

// GenericInterface1 is a dummy interface to test the program output. This
// interface tests generation of generic interfaces with the specified type
// parameters.
type GenericInterface1[Type any] interface {
	// Method1 is the first method of GenericInterface1.
	Method1() Type
	// Method2 is the second method of GenericInterface1.
	Method2(Type)
	// Method3 is the third method of GenericInterface1.
	Method3(Type) Type
}

// GenericInterface2 is a dummy interface to test the program output. This
// interface tests generation of generic interfaces with the specified type
// parameters.
type GenericInterface2[Type1 any, Type2 comparable] interface {
	// Method1 is the first method of GenericInterface2.
	Method1() (Type1, Type2)
	// Method2 is the second method of GenericInterface2.
	Method2(Type1, Type2)
	// Method3 is the third method of GenericInterface2.
	Method3(Type1) Type2
}

// GenericInterface3 is a dummy interface to test the program output. This
// interface tests generation of generic interfaces with repeated type
// parameters.
type GenericInterface3[Type1, Type2 any] interface {
	// Method1 is the first method of GenericInterface3.
	Method1() (Type1, Type2)
	// Method2 is the second method of GenericInterface3.
	Method2(Type1, Type2)
	// Method3 is the third method of GenericInterface3.
	Method3(Type1) Type2
}

// Interface1Output is the expected output generated from reflecting on
// Interface1, provided that the receiver is equal to 'r *Receiver'.
var Interface1Output = `// Method1 is the first method of Interface1.
func (r *Receiver) Method1(arg1 string, arg2 string) (result string, err error) {
	panic("not implemented") // TODO: Implement
}

// Method2 is the second method of Interface1.
func (r *Receiver) Method2(arg1 int, arg2 int) (result int, err error) {
	panic("not implemented") // TODO: Implement
}

// Method3 is the third method of Interface1.
func (r *Receiver) Method3(arg1 bool, arg2 bool) (result bool, err error) {
	panic("not implemented") // TODO: Implement
}

`

// Interface2Output is the expected output generated from reflecting on
// Interface2, provided that the receiver is equal to 'r *Receiver'.
var Interface2Output = `/*
	Method1 is the first method of Interface2.
*/
func (r *Receiver) Method1(arg1 int64, arg2 int64) (result int64, err error) {
	panic("not implemented") // TODO: Implement
}

/*
	Method2 is the second method of Interface2.
*/
func (r *Receiver) Method2(arg1 float64, arg2 float64) (result float64, err error) {
	panic("not implemented") // TODO: Implement
}

/*
	Method3 is the third method of Interface2.
*/
func (r *Receiver) Method3(arg1 interface{}, arg2 interface{}) (result interface{}, err error) {
	panic("not implemented") // TODO: Implement
}

`

// Interface3Output is the expected output generated from reflecting on
// Interface3, provided that the receiver is equal to 'r *Receiver'.
var Interface3Output = `// Method1 is the first method of Interface3.
func (r *Receiver) Method1(_ string, _ string) (string, error) {
	panic("not implemented") // TODO: Implement
}

// Method2 is the second method of Interface3.
func (r *Receiver) Method2(_ int, arg2 int) (_ int, err error) {
	panic("not implemented") // TODO: Implement
}

// Method3 is the third method of Interface3.
func (r *Receiver) Method3(arg1 bool, arg2 bool) (result1 bool, result2 bool) {
	panic("not implemented") // TODO: Implement
}

`

type Implemented struct{}

func (r *Implemented) Method1(arg1 string, arg2 string) (result string, err error) {
	return "", nil
}

// Interface4Output is the expected output generated from reflecting on
// Interface3, provided that the receiver is equal to 'r *Implemented'.
var Interface4Output = `// Method2 is the second method of Interface3.
func (r *Implemented) Method2(_ int, arg2 int) (_ int, err error) {
	panic("not implemented") // TODO: Implement
}

// Method3 is the third method of Interface3.
func (r *Implemented) Method3(arg1 bool, arg2 bool) (result1 bool, result2 bool) {
	panic("not implemented") // TODO: Implement
}

`

type Struct5 struct {
}

type Interface5 interface {
	// Method is the first method of Interface5.
	Method2(arg1 string, arg2 Interface2, arg3 Struct5) (Interface3, error)
}

var Interface5Output = `// Method is the first method of Interface5.
func (r *Implemented) Method2(arg1 string, arg2 Interface2, arg3 Struct5) (Interface3, error) {
	panic("not implemented") // TODO: Implement
}

`

// Interface6Output receiver not in current package
var Interface6Output = `// Method is the first method of Interface5.
func (r *Implemented) Method2(arg1 string, arg2 testdata.Interface2, arg3 testdata.Struct5) (testdata.Interface3, error) {
	panic("not implemented") // TODO: Implement
}

`

type Interface6 interface {
	// Method is the first method of Interface6.
	Method2(arg1 string, arg2 int) (arg3 error)
}

var Interface7Output = `// Method is the first method of Interface6.
func (arg1 *Implemented) Method2(_ string, arg2 int) (arg3 error) {
	panic("not implemented") // TODO: Implement
}

`

var Interface8Output = `// Method is the first method of Interface6.
func (arg3 *Implemented) Method2(arg1 string, arg2 int) (_ error) {
	panic("not implemented") // TODO: Implement
}

`

// GenericInterface1Output is the expected output generated from reflecting on
// GenericInterface1, provided that the receiver is equal to 'r *Receiver' and
// it was generated with the type parameters [string].
var GenericInterface1Output = `// Method1 is the first method of GenericInterface1.
func (r *Receiver) Method1() string {
	panic("not implemented") // TODO: Implement
}

// Method2 is the second method of GenericInterface1.
func (r *Receiver) Method2(_ string) {
	panic("not implemented") // TODO: Implement
}

// Method3 is the third method of GenericInterface1.
func (r *Receiver) Method3(_ string) string {
	panic("not implemented") // TODO: Implement
}

`

// GenericInterface2Output is the expected output generated from reflecting on
// GenericInterface2, provided that the receiver is equal to 'r *Receiver' and
// it was generated with the type parameters [string, bool].
var GenericInterface2Output = `// Method1 is the first method of GenericInterface2.
func (r *Receiver) Method1() (string, bool) {
	panic("not implemented") // TODO: Implement
}

// Method2 is the second method of GenericInterface2.
func (r *Receiver) Method2(_ string, _ bool) {
	panic("not implemented") // TODO: Implement
}

// Method3 is the third method of GenericInterface2.
func (r *Receiver) Method3(_ string) bool {
	panic("not implemented") // TODO: Implement
}

`

// GenericInterface3Output is the expected output generated from reflecting on
// GenericInterface3, provided that the receiver is equal to 'r *Receiver' and
// it was generated with the type parameters [string, bool].
var GenericInterface3Output = `// Method1 is the first method of GenericInterface3.
func (r *Receiver) Method1() (string, bool) {
	panic("not implemented") // TODO: Implement
}

// Method2 is the second method of GenericInterface3.
func (r *Receiver) Method2(_ string, _ bool) {
	panic("not implemented") // TODO: Implement
}

// Method3 is the third method of GenericInterface3.
func (r *Receiver) Method3(_ string) bool {
	panic("not implemented") // TODO: Implement
}

`

type ImplementedGeneric[Type1 any] struct{}

func (r *ImplementedGeneric[Type1]) Method1(arg1 string, arg2 string) (result string, err error) {
	return "", nil
}

var Interface4GenericOutput = `// Method2 is the second method of Interface3.
func (r *ImplementedGeneric[Type1]) Method2(_ int, arg2 int) (_ int, err error) {
	panic("not implemented") // TODO: Implement
}

// Method3 is the third method of Interface3.
func (r *ImplementedGeneric[Type1]) Method3(arg1 bool, arg2 bool) (result1 bool, result2 bool) {
	panic("not implemented") // TODO: Implement
}

`

var Interface5GenericOutput = `// Method is the first method of Interface5.
func (r *ImplementedGeneric[Type1]) Method2(arg1 string, arg2 Interface2, arg3 Struct5) (Interface3, error) {
	panic("not implemented") // TODO: Implement
}

`

// Interface6GenericOutput receiver not in current package
var Interface6GenericOutput = `// Method is the first method of Interface5.
func (r *ImplementedGeneric[Type1]) Method2(arg1 string, arg2 testdata.Interface2, arg3 testdata.Struct5) (testdata.Interface3, error) {
	panic("not implemented") // TODO: Implement
}

`

type ImplementedGenericMultipleParams[Type1 any, Type2 comparable] struct{}

func (r *ImplementedGenericMultipleParams[Type1, Type2]) Method1(arg1 string, arg2 string) (result string, err error) {
	return "", nil
}

var Interface4GenericMultipleParamsOutput = `// Method2 is the second method of Interface3.
func (r *ImplementedGenericMultipleParams[Type1, Type2]) Method2(_ int, arg2 int) (_ int, err error) {
	panic("not implemented") // TODO: Implement
}

// Method3 is the third method of Interface3.
func (r *ImplementedGenericMultipleParams[Type1, Type2]) Method3(arg1 bool, arg2 bool) (result1 bool, result2 bool) {
	panic("not implemented") // TODO: Implement
}

`

var Interface5GenericMultipleParamsOutput = `// Method is the first method of Interface5.
func (r *ImplementedGenericMultipleParams[Type1, Type2]) Method2(arg1 string, arg2 Interface2, arg3 Struct5) (Interface3, error) {
	panic("not implemented") // TODO: Implement
}

`

// Interface6GenericMultipleParamsOutput receiver not in current package
var Interface6GenericMultipleParamsOutput = `// Method is the first method of Interface5.
func (r *ImplementedGenericMultipleParams[Type1, Type2]) Method2(arg1 string, arg2 testdata.Interface2, arg3 testdata.Struct5) (testdata.Interface3, error) {
	panic("not implemented") // TODO: Implement
}

`

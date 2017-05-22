package tester

// An A is a string type
type A string

type d struct {
	c struct{ b struct{} }
}

type c struct {
	a string
	b struct {
		Foo string
		a   struct{}
	}
}

func (c *c) Read(p []byte) (n int, err error) {
	panic("not implemented")
}

type b struct {
	a string
	b string
}

type a func(interface{}) bool

type e interface {
	E() string
}

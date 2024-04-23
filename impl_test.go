package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/josharian/impl/testdata"
)

type errBool bool

func (b errBool) String() string {
	if b {
		return "an error"
	}
	return "no error"
}

func TestFindInterface(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input   string
		path    string
		typ     Type
		wantErr bool
	}{
		{input: "net.Conn", path: "net", typ: Type{Name: "Conn"}},
		{input: "http.ResponseWriter", path: "net/http", typ: Type{Name: "ResponseWriter"}},
		{input: "net.Tennis", wantErr: true},
		{input: "a + b", wantErr: true},
		{input: "t[T,U]", path: "", typ: Type{Name: "t", Params: []string{"T", "U"}}},
		{input: "a/b/c/", wantErr: true},
		{input: "a/b/c/pkg", wantErr: true},
		{input: "a/b/c/pkg.", wantErr: true},
		{input: "a/b/c/pkg.Typ", path: "a/b/c/pkg", typ: Type{Name: "Typ"}},
		{input: "gopkg.in/yaml.v2.Unmarshaler", path: "gopkg.in/yaml.v2", typ: Type{Name: "Unmarshaler"}},
		{input: "github.com/josharian/impl/testdata.GenericInterface1[string]", path: "github.com/josharian/impl/testdata", typ: Type{Name: "GenericInterface1", Params: []string{"string"}}},
		{input: "github.com/josharian/impl/testdata.GenericInterface1[*string]", path: "github.com/josharian/impl/testdata", typ: Type{Name: "GenericInterface1", Params: []string{"*string"}}},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			path, typ, err := findInterface(tt.input, ".")
			gotErr := err != nil
			if tt.wantErr != gotErr {
				t.Fatalf("findInterface(%q).err=%v want %s", tt.input, err, errBool(tt.wantErr))
			}
			if tt.path != path {
				t.Errorf("findInterface(%q).path=%q want %q", tt.input, path, tt.path)
			}
			if tt.typ.Name != typ.Name {
				t.Errorf("findInterface(%q).id=%q want %q", tt.input, typ.Name, tt.typ.Name)
			}
			if len(tt.typ.Params) != len(typ.Params) {
				t.Errorf("findInterface(%q).len(typeParams)=%d want %d", tt.input, len(typ.Params), len(tt.typ.Params))
			}
			for pos, v := range tt.typ.Params {
				if v != typ.Params[pos] {
					t.Errorf("findInterface(%q).typeParams[%d]=%q, want %q", tt.input, pos, typ.Params[pos], v)
				}
			}
		})
	}
}

func TestTypeSpec(t *testing.T) {
	// For now, just test whether we can find the interface.
	cases := []struct {
		path    string
		typ     Type
		wantErr bool
	}{
		{path: "net", typ: Type{Name: "Conn"}},
		{path: "net", typ: Type{Name: "Con"}, wantErr: true},
	}

	for _, tt := range cases {
		p, spec, err := typeSpec(tt.path, tt.typ, "")
		gotErr := err != nil
		if tt.wantErr != gotErr {
			t.Errorf("typeSpec(%q, %q).err=%v want %s", tt.path, tt.typ, err, errBool(tt.wantErr))
			continue
		}
		if err == nil {
			if reflect.DeepEqual(p, Pkg{}) {
				t.Errorf("typeSpec(%q, %q).pkg=Pkg{} want non-nil", tt.path, tt.typ)
			}
			if reflect.DeepEqual(spec, Spec{}) {
				t.Errorf("typeSpec(%q, %q).spec=Spec{} want non-nil", tt.path, tt.typ)
			}
		}
	}
}

func TestFuncs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		iface    string
		comments EmitComments
		want     []Func
		wantErr  bool
	}{
		{
			iface: "io.ReadWriter",
			want: []Func{
				{
					Name:   "Read",
					Params: []Param{{Name: "p", Type: "[]byte"}},
					Res: []Param{
						{Name: "n", Type: "int"},
						{Name: "err", Type: "error"},
					},
				},
				{
					Name:   "Write",
					Params: []Param{{Name: "p", Type: "[]byte"}},
					Res: []Param{
						{Name: "n", Type: "int"},
						{Name: "err", Type: "error"},
					},
				},
			},
			comments: WithComments,
		},
		{
			iface: "http.ResponseWriter",
			want: []Func{
				{
					Name: "Header",
					Res:  []Param{{Type: "http.Header"}},
				},
				{
					Name:   "Write",
					Params: []Param{{Name: "_", Type: "[]byte"}},
					Res:    []Param{{Type: "int"}, {Type: "error"}},
				},
				{
					Name:   "WriteHeader",
					Params: []Param{{Type: "int", Name: "statusCode"}},
				},
			},
			comments: WithComments,
		},
		{
			iface: "http.Handler",
			want: []Func{
				{
					Name: "ServeHTTP",
					Params: []Param{
						{Name: "_", Type: "http.ResponseWriter"},
						{Name: "_", Type: "*http.Request"},
					},
				},
			},
			comments: WithComments,
		},
		{
			iface: "ast.Node",
			want: []Func{
				{
					Name: "Pos",
					Res:  []Param{{Type: "token.Pos"}},
				},
				{
					Name: "End",
					Res:  []Param{{Type: "token.Pos"}},
				},
			},
			comments: WithComments,
		},
		{
			iface: "cipher.AEAD",
			want: []Func{
				{
					Name: "NonceSize",
					Res:  []Param{{Type: "int"}},
				},
				{
					Name: "Overhead",
					Res:  []Param{{Type: "int"}},
				},
				{
					Name: "Seal",
					Params: []Param{
						{Name: "dst", Type: "[]byte"},
						{Name: "nonce", Type: "[]byte"},
						{Name: "plaintext", Type: "[]byte"},
						{Name: "additionalData", Type: "[]byte"},
					},
					Res: []Param{{Type: "[]byte"}},
				},
				{
					Name: "Open",
					Params: []Param{
						{Name: "dst", Type: "[]byte"},
						{Name: "nonce", Type: "[]byte"},
						{Name: "ciphertext", Type: "[]byte"},
						{Name: "additionalData", Type: "[]byte"},
					},
					Res: []Param{{Type: "[]byte"}, {Type: "error"}},
				},
			},
			comments: WithComments,
		},
		{
			iface: "error",
			want: []Func{
				{
					Name: "Error",
					Res:  []Param{{Type: "string"}},
				},
			},
			comments: WithComments,
		},
		{
			iface: "error",
			want: []Func{
				{
					Name: "Error",
					Res:  []Param{{Type: "string"}},
				},
			},
			comments: WithComments,
		},
		{
			iface: "http.Flusher",
			want: []Func{
				{
					Name:     "Flush",
					Comments: "// Flush sends any buffered data to the client.\n",
				},
			},
			comments: WithComments,
		},
		{
			iface: "http.Flusher",
			want: []Func{
				{
					Name: "Flush",
				},
			},
			comments: WithoutComments,
		},
		{
			iface: "net.Listener",
			want: []Func{
				{
					Name: "Accept",
					Res:  []Param{{Type: "net.Conn"}, {Type: "error"}},
				},
				{
					Name: "Close",
					Res:  []Param{{Type: "error"}},
				},
				{
					Name: "Addr",
					Res:  []Param{{Type: "net.Addr"}},
				},
			},
			comments: WithComments,
		},
		{iface: "net.Tennis", wantErr: true},
		{
			iface: "github.com/josharian/impl/testdata.GenericInterface1[int]",
			want: []Func{
				{
					Name: "Method1",
					Res:  []Param{{Type: "int"}},
				},
				{
					Name:   "Method2",
					Params: []Param{{Name: "_", Type: "int"}},
				},
				{
					Name:   "Method3",
					Params: []Param{{Name: "_", Type: "int"}},
					Res:    []Param{{Type: "int"}},
				},
			},
			comments: WithComments,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.iface, func(t *testing.T) {
			t.Parallel()
			fns, err := funcs(tt.iface, "", "", tt.comments)
			gotErr := err != nil
			if tt.wantErr != gotErr {
				t.Fatalf("funcs(%q).err=%v want %s", tt.iface, err, errBool(tt.wantErr))
			}

			if len(fns) != len(tt.want) {
				t.Errorf("funcs(%q).fns=\n%v\nwant\n%v\n", tt.iface, fns, tt.want)
			}
			for i, fn := range fns {
				if fn.Name != tt.want[i].Name ||
					!reflect.DeepEqual(fn.Params, tt.want[i].Params) ||
					!reflect.DeepEqual(fn.Res, tt.want[i].Res) {

					t.Errorf("funcs(%q).fns=\n%v\nwant\n%v\n", tt.iface, fns, tt.want)
				}
				if tt.comments == WithoutComments && fn.Comments != "" {
					t.Errorf("funcs(%q).comments=\n%v\nbut comments disabled", tt.iface, fns)
				}
			}
		})
	}
}

func TestValidReceiver(t *testing.T) {
	cases := []struct {
		recv string
		want bool
	}{
		{recv: "f", want: true},
		{recv: "f[T]", want: true},
		{recv: "f[T, U]", want: true},
		{recv: "F", want: true},
		{recv: "*F[T]", want: true},
		{recv: "*F[T, U]", want: true},
		{recv: "f F", want: true},
		{recv: "f *F", want: true},
		{recv: "f *F[T]", want: true},
		{recv: "f *F[T, U]", want: true},
		{recv: "", want: false},
		{recv: "a+b", want: false},
		{recv: "[T]", want: false},
		{recv: "[T, U]", want: false},
	}

	for _, tt := range cases {
		got := validReceiver(tt.recv)
		if got != tt.want {
			t.Errorf("validReceiver(%q)=%t want %t", tt.recv, got, tt.want)
		}
	}
}

func TestValidMethodComments(t *testing.T) {
	cases := []struct {
		iface string
		want  []Func
	}{
		{
			iface: "github.com/josharian/impl/testdata.Interface1",
			want: []Func{
				{
					Name: "Method1",
					Params: []Param{
						{
							Name: "arg1",
							Type: "string",
						}, {
							Name: "arg2",
							Type: "string",
						},
					},
					Res: []Param{
						{
							Name: "result",
							Type: "string",
						},
						{
							Name: "err",
							Type: "error",
						},
					}, Comments: "// Method1 is the first method of Interface1.\n",
				},
				{
					Name: "Method2",
					Params: []Param{
						{
							Name: "arg1",
							Type: "int",
						},
						{
							Name: "arg2",
							Type: "int",
						},
					},
					Res: []Param{
						{
							Name: "result",
							Type: "int",
						},
						{
							Name: "err",
							Type: "error",
						},
					},
					Comments: "// Method2 is the second method of Interface1.\n",
				},
				{
					Name: "Method3",
					Params: []Param{
						{
							Name: "arg1",
							Type: "bool",
						},
						{
							Name: "arg2",
							Type: "bool",
						},
					},
					Res: []Param{
						{
							Name: "result",
							Type: "bool",
						},
						{
							Name: "err",
							Type: "error",
						},
					},
					Comments: "// Method3 is the third method of Interface1.\n",
				},
			},
		},
		{
			iface: "github.com/josharian/impl/testdata.Interface2",
			want: []Func{
				{
					Name: "Method1",
					Params: []Param{
						{
							Name: "arg1",
							Type: "int64",
						},
						{
							Name: "arg2",
							Type: "int64",
						},
					},
					Res: []Param{
						{
							Name: "result",
							Type: "int64",
						},
						{
							Name: "err",
							Type: "error",
						},
					},
					Comments: "/*\n\t\tMethod1 is the first method of Interface2.\n\t*/\n",
				},
				{
					Name: "Method2",
					Params: []Param{
						{
							Name: "arg1",
							Type: "float64",
						},
						{
							Name: "arg2",
							Type: "float64",
						},
					},
					Res: []Param{
						{
							Name: "result",
							Type: "float64",
						},
						{
							Name: "err",
							Type: "error",
						},
					},
					Comments: "/*\n\t\tMethod2 is the second method of Interface2.\n\t*/\n",
				},
				{
					Name: "Method3",
					Params: []Param{
						{
							Name: "arg1",
							Type: "interface{}",
						},
						{
							Name: "arg2",
							Type: "interface{}",
						},
					},
					Res: []Param{
						{
							Name: "result",
							Type: "interface{}",
						},
						{
							Name: "err",
							Type: "error",
						},
					},
					Comments: "/*\n\t\tMethod3 is the third method of Interface2.\n\t*/\n",
				},
			},
		},
		{
			iface: "github.com/josharian/impl/testdata.Interface3",
			want: []Func{
				{
					Name: "Method1",
					Params: []Param{
						{
							Name: "_",
							Type: "string",
						}, {
							Name: "_",
							Type: "string",
						},
					},
					Res: []Param{
						{
							Name: "",
							Type: "string",
						},
						{
							Name: "",
							Type: "error",
						},
					}, Comments: "// Method1 is the first method of Interface3.\n",
				},
				{
					Name: "Method2",
					Params: []Param{
						{
							Name: "_",
							Type: "int",
						},
						{
							Name: "arg2",
							Type: "int",
						},
					},
					Res: []Param{
						{
							Name: "_",
							Type: "int",
						},
						{
							Name: "err",
							Type: "error",
						},
					},
					Comments: "// Method2 is the second method of Interface3.\n",
				},
				{
					Name: "Method3",
					Params: []Param{
						{
							Name: "arg1",
							Type: "bool",
						},
						{
							Name: "arg2",
							Type: "bool",
						},
					},
					Res: []Param{
						{
							Name: "result1",
							Type: "bool",
						},
						{
							Name: "result2",
							Type: "bool",
						},
					},
					Comments: "// Method3 is the third method of Interface3.\n",
				},
			},
		},
	}

	for _, tt := range cases {
		fns, err := funcs(tt.iface, ".", "", WithComments)
		if err != nil {
			t.Errorf("funcs(%q).err=%v", tt.iface, err)
		}
		if !reflect.DeepEqual(fns, tt.want) {
			t.Errorf("funcs(%q).fns=\n%v\nwant\n%v\n", tt.iface, fns, tt.want)
		}
	}
}

func TestStubGeneration(t *testing.T) {
	cases := []struct {
		iface string
		want  string
		dir   string
	}{
		{
			iface: "github.com/josharian/impl/testdata.Interface1",
			want:  testdata.Interface1Output,
			dir:   ".",
		},
		{
			iface: "github.com/josharian/impl/testdata.Interface2",
			want:  testdata.Interface2Output,
			dir:   ".",
		},
		{
			iface: "github.com/josharian/impl/testdata.Interface3",
			want:  testdata.Interface3Output,
			dir:   ".",
		},
		{
			iface: "Interface1",
			want:  testdata.Interface1Output,
			dir:   "testdata",
		},
		{
			iface: "github.com/josharian/impl/testdata.Interface9",
			want:  testdata.Interface9Output,
			dir:   ".",
		},
		{
			iface: "github.com/josharian/impl/testdata.GenericInterface1[string]",
			want:  testdata.GenericInterface1Output,
			dir:   ".",
		},
		{
			iface: "GenericInterface1[string]",
			want:  testdata.GenericInterface1Output,
			dir:   "testdata",
		},
		{
			iface: "github.com/josharian/impl/testdata.GenericInterface2[string, bool]",
			want:  testdata.GenericInterface2Output,
			dir:   ".",
		},
		{
			iface: "GenericInterface2[string, bool]",
			want:  testdata.GenericInterface2Output,
			dir:   "testdata",
		},
		{
			iface: "github.com/josharian/impl/testdata.GenericInterface3[string, bool]",
			want:  testdata.GenericInterface3Output,
			dir:   ".",
		},
		{
			iface: "GenericInterface3[string, bool]",
			want:  testdata.GenericInterface3Output,
			dir:   "testdata",
		},
	}
	for _, tt := range cases {
		t.Run(tt.iface, func(t *testing.T) {
			fns, err := funcs(tt.iface, tt.dir, "", WithComments)
			if err != nil {
				t.Errorf("funcs(%q).err=%v", tt.iface, err)
			}
			src := genStubs("r *Receiver", fns, nil)
			if string(src) != tt.want {
				t.Errorf("genStubs(\"r *Receiver\", %+#v).src=\n%#v\nwant\n%#v\n", fns, string(src), tt.want)
			}
		})
	}
}

func TestStubGenerationForImplemented(t *testing.T) {
	cases := []struct {
		desc    string
		iface   string
		recv    string
		recvPkg string
		want    string
	}{
		{
			desc:    "without implemeted methods",
			iface:   "github.com/josharian/impl/testdata.Interface3",
			recv:    "r *Implemented",
			recvPkg: "testdata",
			want:    testdata.Interface4Output,
		},
		{
			desc:    "without implemeted methods with trailing space",
			iface:   "github.com/josharian/impl/testdata.Interface3",
			recv:    "r *Implemented ",
			recvPkg: "testdata",
			want:    testdata.Interface4Output,
		},
		{
			desc:    "without implemeted methods, with generic receiver",
			iface:   "github.com/josharian/impl/testdata.Interface3",
			recv:    "r *ImplementedGeneric[Type1]",
			recvPkg: "testdata",
			want:    testdata.Interface4GenericOutput,
		},
		{
			desc:    "without implemeted methods, with generic receiver with multiple params",
			iface:   "github.com/josharian/impl/testdata.Interface3",
			recv:    "r *ImplementedGenericMultipleParams[Type1, Type2]",
			recvPkg: "testdata",
			want:    testdata.Interface4GenericMultipleParamsOutput,
		},
		{
			desc:    "without implemeted methods and receiver variable",
			iface:   "github.com/josharian/impl/testdata.Interface3",
			recv:    "*Implemented",
			recvPkg: "testdata",
			want:    strings.ReplaceAll(testdata.Interface4Output, "r *Implemented", "*Implemented"),
		},
		{
			desc:    "receiver and interface in the same package",
			iface:   "github.com/josharian/impl/testdata.Interface5",
			recv:    "r *Implemented",
			recvPkg: "testdata",
			want:    testdata.Interface5Output,
		},
		{
			desc:    "generic receiver and interface in the same package",
			iface:   "github.com/josharian/impl/testdata.Interface5",
			recv:    "r *ImplementedGeneric[Type1]",
			recvPkg: "testdata",
			want:    testdata.Interface5GenericOutput,
		},
		{
			desc:    "generic receiver with multiple params and interface in the same package",
			iface:   "github.com/josharian/impl/testdata.Interface5",
			recv:    "r *ImplementedGenericMultipleParams[Type1, Type2]",
			recvPkg: "testdata",
			want:    testdata.Interface5GenericMultipleParamsOutput,
		},
		{
			desc:    "receiver and interface in a different package",
			iface:   "github.com/josharian/impl/testdata.Interface5",
			recv:    "r *Implemented",
			recvPkg: "test",
			want:    testdata.Interface6Output,
		},
		{
			desc:    "generic receiver and interface in a different package",
			iface:   "github.com/josharian/impl/testdata.Interface5",
			recv:    "r *ImplementedGeneric[Type1]",
			recvPkg: "test",
			want:    testdata.Interface6GenericOutput,
		},
		{
			desc:    "generic receiver with multiple params and interface in a different package",
			iface:   "github.com/josharian/impl/testdata.Interface5",
			recv:    "r *ImplementedGenericMultipleParams[Type1, Type2]",
			recvPkg: "test",
			want:    testdata.Interface6GenericMultipleParamsOutput,
		},
	}
	for _, tt := range cases {
		t.Run(tt.desc, func(t *testing.T) {
			fns, err := funcs(tt.iface, ".", tt.recvPkg, WithComments)
			if err != nil {
				t.Errorf("funcs(%q).err=%v", tt.iface, err)
			}

			implemented, err := implementedFuncs(fns, tt.recv, "testdata")
			if err != nil {
				t.Errorf("ifuncs.err=%v", err)
			}
			src := genStubs(tt.recv, fns, implemented)
			if string(src) != tt.want {
				t.Errorf("genStubs(\"r *Implemented\", %+#v).src=\n\n%#v\n\nwant\n\n%#v\n\n", fns, string(src), tt.want)
			}
		})
	}
}

func TestStubGenerationForRepeatedName(t *testing.T) {
	cases := []struct {
		desc    string
		iface   string
		recv    string
		recvPkg string
		want    string
	}{
		{
			desc:    "receiver and in.Params with the same name",
			iface:   "github.com/josharian/impl/testdata.Interface6",
			recv:    "arg1 *Implemented",
			recvPkg: "testdata",
			want:    testdata.Interface7Output,
		},
		{
			desc:    "receiver and out.Params with the same name",
			iface:   "github.com/josharian/impl/testdata.Interface6",
			recv:    "arg3 *Implemented",
			recvPkg: "testdata",
			want:    testdata.Interface8Output,
		},
	}
	for _, tt := range cases {
		t.Run(tt.desc, func(t *testing.T) {
			fns, err := funcs(tt.iface, ".", tt.recvPkg, WithComments)
			if err != nil {
				t.Errorf("funcs(%q).err=%v", tt.iface, err)
			}

			implemented, err := implementedFuncs(fns, tt.recv, "testdata")
			if err != nil {
				t.Errorf("ifuncs.err=%v", err)
			}
			src := genStubs(tt.recv, fns, implemented)
			if string(src) != tt.want {
				t.Errorf("genStubs(\"r *Implemented\", %+#v).src=\n\n%#v\n\nwant\n\n%#v\n\n", fns, string(src), tt.want)
			}
		})
	}
}

func TestParseTypeParams(t *testing.T) {
	t.Parallel()

	cases := []struct {
		desc    string
		input   string
		want    Type
		wantErr bool
	}{
		{desc: "non-generic type", input: "Reader", want: Type{Name: "Reader"}},
		{desc: "one type param", input: "Reader[Foo]", want: Type{Name: "Reader", Params: []string{"Foo"}}},
		{desc: "two type params", input: "Reader[Foo, Bar]", want: Type{Name: "Reader", Params: []string{"Foo", "Bar"}}},
		{desc: "three type params", input: "Reader[Foo, Bar, Baz]", want: Type{Name: "Reader", Params: []string{"Foo", "Bar", "Baz"}}},
		{desc: "no spaces", input: "Reader[Foo,Bar]", want: Type{Name: "Reader", Params: []string{"Foo", "Bar"}}},
		{desc: "unclosed brackets", input: "Reader[Foo", wantErr: true},
		{desc: "no params", input: "Reader[]", wantErr: true},
		{desc: "space-only params", input: "Reader[ ]", wantErr: true},
		{desc: "multiple space-only params", input: "Reader[ , , ]", wantErr: true},
		{desc: "characters after bracket", input: "Reader[Foo]Bar", wantErr: true},
		{desc: "qualified generic type", input: "io.Reader[Foo]", want: Type{Name: "io.Reader", Params: []string{"Foo"}}},
		{desc: "qualified generic type with two params", input: "io.Reader[Foo, Bar]", want: Type{Name: "io.Reader", Params: []string{"Foo", "Bar"}}},
		{desc: "qualified generic param", input: "Reader[io.Reader]", want: Type{Name: "Reader", Params: []string{"io.Reader"}}},
		{desc: "qualified and unqualified generic param", input: "Reader[io.Reader, string]", want: Type{Name: "Reader", Params: []string{"io.Reader", "string"}}},
		{desc: "pointer qualified generic param", input: "Reader[*io.Reader]", want: Type{Name: "Reader", Params: []string{"*io.Reader"}}},
		{desc: "map generic param", input: "Reader[map[string]string]", want: Type{Name: "Reader", Params: []string{"map[string]string"}}},
		{desc: "pointer map generic param", input: "Reader[*map[string]string]", want: Type{Name: "Reader", Params: []string{"*map[string]string"}}},
		{desc: "pointer key map generic param", input: "Reader[map[*string]string]", want: Type{Name: "Reader", Params: []string{"map[*string]string"}}},
		{desc: "pointer value map generic param", input: "Reader[map[string]*string]", want: Type{Name: "Reader", Params: []string{"map[string]*string"}}},
		{desc: "slice generic param", input: "Reader[[]string]", want: Type{Name: "Reader", Params: []string{"[]string"}}},
		{desc: "pointer slice generic param", input: "Reader[*[]string]", want: Type{Name: "Reader", Params: []string{"*[]string"}}},
		{desc: "pointer slice value generic param", input: "Reader[[]*string]", want: Type{Name: "Reader", Params: []string{"[]*string"}}},
		{desc: "array generic param", input: "Reader[[1]string]", want: Type{Name: "Reader", Params: []string{"[1]string"}}},
		{desc: "pointer array generic param", input: "Reader[*[1]string]", want: Type{Name: "Reader", Params: []string{"*[1]string"}}},
		{desc: "pointer array value generic param", input: "Reader[[1]*string]", want: Type{Name: "Reader", Params: []string{"[1]*string"}}},
		{desc: "chan generic param", input: "Reader[chan error]", want: Type{Name: "Reader", Params: []string{"chan error"}}},
		{desc: "receiver chan generic param", input: "Reader[<-chan error]", want: Type{Name: "Reader", Params: []string{"<-chan error"}}},
		{desc: "send chan generic param", input: "Reader[chan<- error]", want: Type{Name: "Reader", Params: []string{"chan<- error"}}},
		{desc: "pointer chan generic param", input: "Reader[*chan error]", want: Type{Name: "Reader", Params: []string{"*chan error"}}},
		{desc: "func generic param", input: "Reader[func() string]", want: Type{Name: "Reader", Params: []string{"func() string"}}},
		{desc: "one arg func generic param", input: "Reader[func(a int) string]", want: Type{Name: "Reader", Params: []string{"func(a int) string"}}},
		{desc: "two arg one type func generic param", input: "Reader[func(a, b int) string]", want: Type{Name: "Reader", Params: []string{"func(a, b int) string"}}},
		{desc: "three arg one type func generic param", input: "Reader[func(a, b, c int) string]", want: Type{Name: "Reader", Params: []string{"func(a, b, c int) string"}}},
		{desc: "three arg two types func generic param", input: "Reader[func(a, b string, c int) string]", want: Type{Name: "Reader", Params: []string{"func(a, b string, c int) string"}}},
		{desc: "three arg three types func generic param", input: "Reader[func(a bool, b string, c int) string]", want: Type{Name: "Reader", Params: []string{"func(a bool, b string, c int) string"}}},
		// don't need support for generics on the function type itself; function types must have no type parameters
		// https://cs.opensource.google/go/go/+/master:src/go/parser/parser.go;l=1048;drc=cafb49ac731f862f386862d64b27b8314eeb2909
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			typ, err := parseType(tt.input)
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Fatalf("unexpected error: %s", err)
			}
			if typ.Name != tt.want.Name {
				t.Errorf("wanted ID %q, got %q", tt.want.Name, typ.Name)
			}
			if len(typ.Params) != len(tt.want.Params) {
				t.Errorf("wanted %d params, got %d: %v", len(tt.want.Params), len(typ.Params), typ.Params)
			}
			for pos, param := range typ.Params {
				if param != tt.want.Params[pos] {
					t.Errorf("expected param %d to be %q, got %q: %v", pos, tt.want.Params[pos], param, typ.Params)
				}
			}
		})
	}
}

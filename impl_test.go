package main

import (
	"reflect"
	"testing"
)

type errBool bool

func (b errBool) String() string {
	if b {
		return "an error"
	}
	return "no error"
}

func TestFindInterface(t *testing.T) {
	cases := []struct {
		iface   string
		path    string
		id      string
		wantErr bool
	}{
		{iface: "net.Conn", path: "net", id: "Conn"},
		{iface: "http.ResponseWriter", path: "net/http", id: "ResponseWriter"},
		{iface: "net.Tennis", wantErr: true},
		{iface: "a + b", wantErr: true},
		{iface: "a/b/c/", wantErr: true},
		{iface: "a/b/c/pkg", wantErr: true},
		{iface: "a/b/c/pkg.", wantErr: true},
		{iface: "a/b/c/pkg.Typ", path: "a/b/c/pkg", id: "Typ"},
		{iface: "a/b/c/pkg.Typ.Foo", wantErr: true},
	}

	for _, tt := range cases {
		path, id, err := findInterface(tt.iface, ".")
		gotErr := err != nil
		if tt.wantErr != gotErr {
			t.Errorf("findInterface(%q).err=%v want %s", tt.iface, err, errBool(tt.wantErr))
			continue
		}
		if tt.path != path {
			t.Errorf("findInterface(%q).path=%q want %q", tt.iface, path, tt.path)
		}
		if tt.id != id {
			t.Errorf("findInterface(%q).id=%q want %q", tt.iface, id, tt.id)
		}
	}
}

func TestTypeSpec(t *testing.T) {
	// For now, just test whether we can find the interface.
	cases := []struct {
		path    string
		id      string
		wantErr bool
	}{
		{path: "net", id: "Conn"},
		{path: "net", id: "Con", wantErr: true},
	}

	for _, tt := range cases {
		p, spec, err := typeSpec(tt.path, tt.id, "")
		gotErr := err != nil
		if tt.wantErr != gotErr {
			t.Errorf("typeSpec(%q, %q).err=%v want %s", tt.path, tt.id, err, errBool(tt.wantErr))
			continue
		}
		if err == nil {
			if reflect.DeepEqual(p, Pkg{}) {
				t.Errorf("typeSpec(%q, %q).pkg=Pkg{} want non-nil", tt.path, tt.id)
			}
			if reflect.DeepEqual(spec, Spec{}) {
				t.Errorf("typeSpec(%q, %q).spec=Spec{} want non-nil", tt.path, tt.id)
			}
		}
	}
}

func TestFuncs(t *testing.T) {
	cases := []struct {
		iface          string
		want           []Func
		wantErr        bool
		ignoreComments bool
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
					Params: []Param{{Type: "[]byte"}},
					Res:    []Param{{Type: "int"}, {Type: "error"}},
				},
				{
					Name:   "WriteHeader",
					Params: []Param{{Type: "int", Name: "statusCode"}},
				},
			},
			ignoreComments: true,
		},
		{
			iface: "http.Handler",
			want: []Func{
				{
					Name:   "ServeHTTP",
					Params: []Param{{Type: "http.ResponseWriter"}, {Type: "*http.Request"}},
				},
			},
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
			ignoreComments: true,
		},
		{
			iface: "error",
			want: []Func{
				{
					Name: "Error",
					Res:  []Param{{Type: "string"}},
				},
			},
		},
		{
			iface: "error",
			want: []Func{
				{
					Name: "Error",
					Res:  []Param{{Type: "string"}},
				},
			},
		},
		{
			iface: "http.Flusher",
			want: []Func{
				{
					Name:     "Flush",
					Comments: "// Flush sends any buffered data to the client.\n",
				},
			},
		},
		{
			iface: "net.Listener",
			want: []Func{
				{
					Name:     "Accept",
					Comments: "// Accept waits for and returns the next connection to the listener.\n",
					Res:      []Param{{Type: "net.Conn"}, {Type: "error"}},
				},
				{
					Name:     "Close",
					Comments: "// Close closes the listener.\n// Any blocked Accept operations will be unblocked and return errors.\n",
					Res:      []Param{{Type: "error"}},
				},
				{
					Name:     "Addr",
					Comments: "// Addr returns the listener's network address.\n",
					Res:      []Param{{Type: "net.Addr"}},
				},
			},
		},
		{iface: "net.Tennis", wantErr: true},
	}

	for _, tt := range cases {
		fns, err := funcs(tt.iface, "")
		gotErr := err != nil
		if tt.wantErr != gotErr {
			t.Errorf("funcs(%q).err=%v want %s", tt.iface, err, errBool(tt.wantErr))
			continue
		}

		if tt.ignoreComments {
			if len(fns) != len(tt.want) {
				t.Errorf("funcs(%q).fns=\n%v\nwant\n%v\n", tt.iface, fns, tt.want)
			}
			for i, fn := range fns {
				if fn.Name != tt.want[i].Name ||
					!reflect.DeepEqual(fn.Params, tt.want[i].Params) ||
					!reflect.DeepEqual(fn.Res, tt.want[i].Res) {

					t.Errorf("funcs(%q).fns=\n%v\nwant\n%v\n", tt.iface, fns, tt.want)
				}
			}
		} else {
			if !reflect.DeepEqual(fns, tt.want) {
				t.Errorf("funcs(%q).fns=\n%v\nwant\n%v\n", tt.iface, fns, tt.want)
			}
		}

	}
}

func TestValidReceiver(t *testing.T) {
	cases := []struct {
		recv string
		want bool
	}{
		{recv: "f", want: true},
		{recv: "F", want: true},
		{recv: "f F", want: true},
		{recv: "f *F", want: true},
		{recv: "", want: false},
		{recv: "a+b", want: false},
	}

	for _, tt := range cases {
		got := validReceiver(tt.recv)
		if got != tt.want {
			t.Errorf("validReceiver(%q)=%t want %t", tt.recv, got, tt.want)
		}
	}
}

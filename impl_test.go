package main

import (
	"reflect"
	"testing"
)

type errBool bool

func (b errBool) String() string {
	if b {
		return "an error"
	} else {
		return "no error"
	}
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
	}

	for _, tt := range cases {
		path, id, err := findInterface(tt.iface)
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
		p, spec, err := typeSpec(tt.path, tt.id)
		gotErr := err != nil
		if tt.wantErr != gotErr {
			t.Errorf("typeSpec(%q, %q).err=%v want %s", tt.path, tt.id, err, errBool(tt.wantErr))
			continue
		}
		if err == nil {
			if reflect.DeepEqual(p, Pkg{}) {
				t.Errorf("typeSpec(%q, %q).pkg=Pkg{} want non-nil", tt.path, tt.id)
			}
			if spec == nil {
				t.Errorf("typeSpec(%q, %q).spec=nil want non-nil", tt.path, tt.id)
			}
		}
	}
}

func TestFuncs(t *testing.T) {
	cases := []struct {
		iface   string
		want    []Func
		wantErr bool
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
					Params: []Param{{Type: "int"}},
				},
			},
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
		{iface: "net.Tennis", wantErr: true},
	}

	for _, tt := range cases {
		fns, err := funcs(tt.iface)
		gotErr := err != nil
		if tt.wantErr != gotErr {
			t.Errorf("funcs(%q).err=%v want %s", tt.iface, err, errBool(tt.wantErr))
			continue
		}
		if !reflect.DeepEqual(fns, tt.want) {
			t.Errorf("funcs(%q).fns=%q want %q", tt.iface, fns, tt.want)
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

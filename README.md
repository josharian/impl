`impl` generates method stubs for implementing an interface.

```bash
go get -u github.com/josharian/impl
```

Sample usage:

```bash
$ impl 'f *File' io.ReadWriteCloser
func (f *File) Read(p []byte) (n int, err error) {
	panic("not implemented")
}

func (f *File) Write(p []byte) (n int, err error) {
	panic("not implemented")
}

func (f *File) Close() error {
	panic("not implemented")
}

# You can also provide a full name by specifying the package path.
# This helps in cases where the interface can't be guessed
# just from the package name and interface name.
$ impl 's *Source' golang.org/x/oauth2.TokenSource
func (s *Source) Token() (*oauth2.Token, error) {
    panic("not implemented")
}

# It is possible to set a custom method body.     
# Body will be interpreted as a text/template, allowing access to fields of the
# Method struct:
#    
#    // Method represents a method signature.
#    type Method struct {
#        Recv Recv
#        Func
#    }
#    
# This way, an user will be able to write a custom body implementation that
# makes use of Recv and Func fields
#    
#    For example:
#    
#    // custom body:
#    defer func() {
#	{{.Recv.Name}}.logger.Log("service", "{{.Name}}", {{range $i, $e := .Params}}{{if gt $i 0}}"{{$e.Name}}", fmt.Sprintf("%+v",{{$e.Name}}), {{end}}{{end}})
#	}();
#	return {{.Recv.Name}}.next.{{.Name}}({{range .Params}}{{.Name}}, {{end}});
#
#	will generate the following output:
$ impl -body $GOPATH/src/github.com/filewalkwithme/stringsvc3/logging/impl.go 'mw *logmw' github.com/filewalkwithme/stringsvc3.StringService
func (mw *logmw) Uppercase(text string) (string, error) {
	defer func() {
		mw.logger.Log("service", "Uppercase")
	}()
	return mw.Uppercase(text)
}

func (mw *logmw) Count(text string) int {
	defer func() {
		mw.logger.Log("service", "Count")
	}()
	return mw.Count(text)
}
```

You can use `impl` from Vim with [vim-go-impl](https://github.com/rhysd/vim-go-impl)

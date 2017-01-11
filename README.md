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

# You can request that the declaring file be updated in-place by passing -u
$ impl -u 'f *File' io.ReadWriteCloser

# You can also specify a position at which you'd like the generated code to be
# inserted, helpful for editor/ide integration.
$ impl -p main.go:12 'f *File' io.ReadWriteCloser

# Finally, you can override the default stdout printing and send your output to 
# a specific file with the -o option.
$ impl -p main.go:12 -o test/main2.go 'f *File' io.ReadWriteCloser
```

You can use `impl` from Vim with [vim-go-impl](https://github.com/rhysd/vim-go-impl)

`impl` generates method stubs for implementing an interface.

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

```

You can use `impl` from Vim with [vim-go-impl](https://github.com/rhysd/vim-go-impl)

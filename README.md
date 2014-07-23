`impl` generates method stubs for implementing an interface.

Sample usage:

```bash
$ impl 'f *File' io.ReadWriteCloser
func (f *File) Read(p []byte) (n int, err error) {
}

func (f *File) Write(p []byte) (n int, err error) {
}

func (f *File) Close() error {
}

```

You can use `impl` from Vim with [vim-go-impl](https://github.com/rhysd/vim-go-impl)

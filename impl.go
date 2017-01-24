// impl generates method stubs for implementing an interface.
package main

import (
	"flag"
	"fmt"
	"go/token"
	"log"
	"os"

	impl "github.com/josharian/impl/pkg"
)

const usage = `impl <recv> <iface>

impl generates method stubs for recv to implement iface.

Examples:

impl 'f *File' io.Reader
impl Murmur hash.Hash

Don't forget the single quotes around the receiver type
to prevent shell globbing.
`

var (
	pos    = flag.String("position", "", "the file:line:col to write the source code to. Default is immediately after the type definition")
	out    = flag.String("o", "", "the file to write out to. default is stdout")
	update = flag.Bool("u", false, "update the file given")
)

func main() {
	flag.Parse()

	imp := impl.Implementer{
		Recv:  flag.Arg(0),
		IFace: flag.Arg(1),
	}

	var p *token.Position

	if *pos != "" {
		// p = &token.Position{}
		fmt.Sscanf(*pos, "%s:%d:%d", &p.Filename, &p.Line, &p.Column)
	}

	bs, err := imp.GenForPosition(p)
	if err != nil {
		log.Fatal(err)
	}

	if *out != "" && *update {
		log.Fatal("Please specify only -u (update in-place) or -o (output file).")
	}

	if *out == "-" || *out == "" {
		*out = "/dev/stdout"
	}

	if *update {
		*out = imp.fset.Position(imp.typeDecl.End()).Filename
	}

	mode := os.O_RDWR | os.O_CREATE

	if f, err := os.Stat(*out); err == nil {
		if f.Mode().IsRegular() {
			mode |= os.O_TRUNC
		}
	}

	f, err := os.OpenFile(*out, mode, 0640)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err = f.Write(bs)
	if err != nil {
		log.Fatal(err)
	}
}

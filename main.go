// impl generates method stubs for implementing an interface.
package main

import (
	"flag"
	"io"
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
	update = flag.Bool("u", false, "update the file in-place (see -p for file defaulting behavior)")
	out    = flag.String("o", "", "the file to write out to. default is stdout")
	pos    = flag.String("p", "", "the file:line[:col] to write the source code to. Default is immediately after the type definition")

	modified = flag.Bool("modified", false, "if files have been modified and not saved, -modified allows consumers to pass guru's archive format on stdin to overlay the directory")
)

func main() {
	flag.Parse()

	imp := impl.Implementer{
		Recv:  flag.Arg(0),
		IFace: flag.Arg(1),
	}

	if *out != "" && *update {
		log.Fatal("Please specify only -u (update in-place) or -o (output file).")
	}

	if *modified {
		imp.Archive = os.Stdin
	}

	var bs []byte
	var err error

	mode := os.O_RDWR | os.O_CREATE

	if *update {
		p, err := imp.Position()
		if err != nil {
			log.Fatal(err)
		}

		*out = p.Filename

		if *pos == "" {
			*pos = p.String()
		}
	}

	if *pos == "" {
		bs, err = imp.GenStubs()
		if err != nil {
			log.Fatal("Error generating stubs:", err)
		}
	} else {
		bs, err = imp.GenForPosition(*pos)
		if err != nil {
			log.Fatal("error generating for position:", err)
		}
	}

	var outFile io.Writer

	if *out != "" && *out != "-" {
		f, err := os.OpenFile(*out, mode, 0640)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		outFile = f
	} else {
		outFile = os.Stdout
	}

	outFile.Write(bs)
}

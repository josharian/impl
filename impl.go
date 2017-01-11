// impl generates method stubs for implementing an interface.
package main

import (
	"flag"
	"fmt"
	"go/token"
	"log"
	"os"
	"strconv"
	"strings"

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
	update = flag.Bool("u", false, "update the file (see -p for file defaulting behavior)")
	out    = flag.String("o", "", "the file to write out to. default is stdout")
	pos    = flag.String("p", "", "the file:line[:col] to write the source code to. Default is immediately after the type definition")
)

func getPosition(pos string) (*token.Position, error) {
	arr := strings.Split(pos, ":")

	if len(arr) < 2 {
		return nil, fmt.Errorf("Invalid position spec")
	}

	p := token.Position{Column: 1}

	p.Filename = arr[0]

	line, err := strconv.Atoi(arr[1])
	if err != nil {
		return nil, fmt.Errorf("invalid line spec in position: %s", err)
	}
	p.Line = line

	if len(arr) == 3 {
		col, err := strconv.Atoi(arr[2])
		if err != nil {
			return nil, fmt.Errorf("invalid column spec in position: %s", err)
		}
		p.Column = col
	}

	return &p, nil
}

func main() {
	flag.Parse()

	imp := impl.Implementer{
		Recv:  flag.Arg(0),
		IFace: flag.Arg(1),
	}

	var p *token.Position

	if *pos != "" {
		var err error
		p, err = getPosition(*pos)
		if err != nil {
			log.Fatal(err)
		}
	}

	// If no options are specified, behave as we would have before, printing out
	// only the generated methods (minus implemented methods)
	if *out == "" && !*update && *pos == "" {
		bs, err := imp.GenStubs()
		if err != nil {
			log.Fatal(err)
		}
		os.Stdout.Write(bs)
		os.Exit(0)
		return
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
		p, err := imp.Position()
		if err != nil {
			log.Fatal(err)
		}
		*out = p.Filename
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

package binbump_test

import (
	"bufio"
	"bytes"
	"fmt"

	"github.com/bengarrett/binbump"
	"golang.org/x/text/encoding/charmap"
)

// 0x41, 0x00, 0x42, 0x08, 0x63, 0x01, 0x64, 0x09, 0x65, 0x02, 0x66, 0x0A, 0x20, 0x07, 0x20, 0x07,

func ExampleBuffer() {
	data := []byte{0x41, 0x00, 0x42, 0x08}
	const cga = binbump.StandardCGA // cga palette
	const width = 160               // columns
	charset := charmap.CodePage437
	r := bytes.NewReader(data)
	buf, _ := binbump.Buffer(r, width, 0, cga, charset)
	fmt.Printf("%q", buf.String())
	// Output: "<div><span style=\"color:#000;background-color:#000;\">A</span><span style=\"color:#555;background-color:#000;\">B</span>\n</div>"
}

func ExampleBuffer_palette() {
	data := []byte{0x41, 0x00, 0x42, 0x08}
	const cga = binbump.RevisedCGA // revised cga palette
	const width = 160              // columns
	charset := charmap.CodePage437
	r := bytes.NewReader(data)
	buf, _ := binbump.Buffer(r, width, 0, cga, charset)
	fmt.Printf("%q", buf.String())
	// Output: "<div><span style=\"color:#000;background-color:#000;\">A</span><span style=\"color:#4e4e4e;background-color:#000;\">B</span>\n</div>"
}

func ExampleBytes() {
	data := []byte{0x41, 0x00, 0x42, 0x08}
	r := bytes.NewReader(data)
	p, _ := binbump.Bytes(r)
	fmt.Printf("%q", p)
	// Output: "<div><span style=\"color:#000;background-color:#000;\">A</span><span style=\"color:#555;background-color:#000;\">B</span>\n</div>"
}

func ExampleString() {
	data := []byte{0x41, 0x00, 0x42, 0x08}
	r := bytes.NewReader(data)
	s, _ := binbump.String(r)
	fmt.Printf("%q", s)
	// Output: "<div><span style=\"color:#000;background-color:#000;\">A</span><span style=\"color:#555;background-color:#000;\">B</span>\n</div>"
}

func ExampleWriteTo() {
	data := []byte{0x41, 0x00, 0x42, 0x08}
	input := bytes.NewReader(data)
	var b bytes.Buffer
	output := bufio.NewWriter(&b)
	cnt, _ := binbump.WriteTo(input, output)
	output.Flush()
	fmt.Printf("%d bytes written\n%q", cnt, b.String())
	// Output: 124 bytes written
	// "<div><span style=\"color:#000;background-color:#000;\">A</span><span style=\"color:#555;background-color:#000;\">B</span>\n</div>"
}

// func TestBuffer_Open(t *testing.T) {
// 	t.Parallel()
// 	file, err := os.Open("testdata/file.bin")
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer file.Close()
//
// 	buf, err := binbump.Buffer(
// 		file, 80, 25, binbump.StandardCGA, nil)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	f, err := os.OpenFile("test.html", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer f.Close()
// 	n, err := io.Copy(f, buf)
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Println("wrote", n, "bytes to test.html")
// }

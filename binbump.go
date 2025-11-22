// Package binbump converts binary screen dumps of the IBM PC graphic and BIOS
// text mode characters, and CGA, EGA, and VGA colors into a HTML representation.
package binbump

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"os"
	"slices"

	"golang.org/x/text/encoding/charmap"
)

var (
	ErrAttribute = errors.New("attribute is not a 4-bit color value")
	ErrReader    = errors.New("reader is nil")
)

// Palette sets the 4-bit (0-15) color codes to a colorset of RGB values.
type Palette uint

const (
	// StandardCGA is the Color Graphics Adapter colorset defined by IBM for the PC in 1981.
	StandardCGA Palette = iota
	// RevisedCGA is the Revised Color Graphics colorset as documented by VilaR,
	// https://int10h.org/blog/2022/06/ibm-5153-color-true-cga-palette/
	RevisedCGA
)

// Color code represented as a hexadecimal triplet or six-digit value.
type Color string

const (
	Black    Color = "000" // 00 black
	Blue     Color = "00a" // 01 blue
	Green    Color = "0a0" // 02 green
	Cyan     Color = "0aa" // 03 cyan
	Red      Color = "a00" // 04 red
	Magenta  Color = "a0a" // 05 magenta
	Brown    Color = "a50" // 06 brown
	Gray     Color = "aaa" // 07 gray
	BlackI   Color = "555" // 08 intense black
	BlueI    Color = "55f" // 09 intense blue
	GreenI   Color = "5f5" // 10 intense green
	CyanI    Color = "5ff" // 11 intense cyan
	RedI     Color = "f55" // 12 intense red
	MagentaI Color = "f5f" // 13 intense magenta
	Yellow   Color = "ff5" // 14 intense brown (yellow)
	White    Color = "fff" // 15 intense gray (white)

	BlueR     Color = "0000c4" // 01 blue
	GreenR    Color = "00c400" // 02 green
	CyanR     Color = "00c4c4" // 03 cyan
	RedR      Color = "c40000" // 04 red
	MagentaR  Color = "c400c4" // 05 magenta
	BrownR    Color = "c47e00" // 06 brown
	GrayR     Color = "c4c4c4" // 07 gray
	BlackIR   Color = "4e4e4e" // 08 intense black
	BlueIR    Color = "4e4edc" // 09 intense blue
	GreenIR   Color = "4edc4e" // 10 intense green
	CyanIR    Color = "4ef3f3" // 11 intense cyan
	RedIR     Color = "dc4e4e" // 12 intense red
	MagentaIR Color = "f34ef3" // 13 intense magenta
	YellowR   Color = "f3f34e" // 14 intense brown (yellow)
)

// BG returns the CSS background-color property and color value.
func (c Color) BG() string {
	if c == "" {
		return ""
	}
	return "background-color:#" + string(c) + ";"
}

// FG returns the CSS color property and color value.
func (c Color) FG() string {
	if c == "" {
		return ""
	}
	return "color:#" + string(c) + ";"
}

type Colors [16]Color

func CGA() Colors {
	return Colors{
		Black, Blue, Green, Cyan, Red, Magenta, Brown, Gray,
		BlackI, BlueI, GreenI, CyanI, RedI, MagentaI, Yellow, White,
	}
}

func CGARevised() Colors {
	return Colors{
		Black, BlueR, GreenR, CyanR, RedR, MagentaR, BrownR, GrayR,
		BlackIR, BlueIR, GreenIR, CyanIR, RedIR, MagentaIR, YellowR, White,
	}
}

// Decoder maintains the screen buffer and print character state.
type Decoder struct {
	Debug       bool // Debug will wrap every character in its own <span> element with a data-xy attribute.
	charset     *charmap.Charmap
	colors      Colors
	columns     int // maximum
	column      int
	row         int
	maxRows     int
	buffer      []template.HTML
	currentLine template.HTML
	currentAttr byte
}

// NewDecoder creates a Decoder with a given width (columns). If width <= 0, 160 is used.
// maxRows should usually be left at 0, its use is only intended for screen dumps that
// contain tailing NULL or corrupt SAUCE metadata that should be ignored.
//
// Palette can either be [StandardCGA] or [RevisedCGA].
//
// Generally the charset of a binary screen dump is [charmap.CodePage437],
// which is used by default when a nil value is used.
func NewDecoder(width, maxRows int, pal Palette, charset *charmap.Charmap) *Decoder {
	if width <= 0 {
		width = 160
	}
	if charset == nil {
		charset = charmap.CodePage437
	}
	d := &Decoder{
		charset: charset,
		columns: width,
		column:  1,
		row:     1,
		maxRows: 0,
	}
	if maxRows > 0 {
		d.maxRows = maxRows
	}
	switch pal {
	case StandardCGA:
		d.colors = CGA()
	case RevisedCGA:
		d.colors = CGARevised()
	default:
		d.colors = CGA()
	}
	return d
}

// Buffer creates a new Buffer containing the HTML elements of the binary dump
// found in the Reader.
//
// The other arguments are used by the [NewDecoder] which documents their purpose.
func Buffer(r io.Reader, width, maxRows int, pal Palette, charset *charmap.Charmap) (*bytes.Buffer, error) {
	if r == nil {
		return nil, ErrReader
	}
	if charset == nil {
		charset = charmap.CodePage437
	}
	d := NewDecoder(width, maxRows, pal, charset)
	if err := d.Read(r); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	out := bufio.NewWriter(&b)
	if err := d.Write(out); err != nil {
		return nil, err
	}
	if err := out.Flush(); err != nil {
		return nil, fmt.Errorf("buffer out flash: %w", err)
	}
	return &b, nil
}

// Bytes returns the HTML elements of the binary dump found in the Reader.
// It assumes the Reader is using IBM Code Page 437 encoding.
func Bytes(r io.Reader) ([]byte, error) {
	buf, err := Buffer(r, 0, 0, StandardCGA, nil)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// String returns the HTML elements of the binary dump found in the Reader.
// It assumes the Reader is using IBM Code Page 437 encoding.
func String(r io.Reader) (string, error) {
	buf, err := Buffer(r, 0, 0, StandardCGA, nil)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WriteTo writes to w the HTML elements of the binary dump found in the Reader.
// It assumes the Reader is using IBM Code Page 437 encoding.
// If width is <= 0, an 80 columns value is used.
//
// The return int64 is the number of bytes written.
func WriteTo(r io.Reader, w io.Writer) (int64, error) {
	buf, err := Buffer(r, 0, 0, StandardCGA, nil)
	if err != nil {
		return 0, err
	}
	i, err := buf.WriteTo(w)
	if err != nil {
		return 0, fmt.Errorf("buffer write to: %w", err)
	}
	return i, nil
}

// Write writes to w the full HTML fragment with outer div and inner lines joined with newlines.
func (d *Decoder) Write(wr io.Writer) error {
	if wr == nil {
		wr = io.Discard
	}
	t, err := template.New("dump").Parse(
		`{{define "T"}}<div>{{ . }}</div>{{end}}`)
	if err != nil {
		return fmt.Errorf("write template parse: %w", err)
	}
	var data template.HTML
	for s := range slices.Values(d.buffer) {
		data += s
	}
	if err := t.ExecuteTemplate(wr, "T", data); err != nil {
		return fmt.Errorf("write template execute: %w", err)
	}
	return nil
}

// Read reads each pair of bytes from r and interprets the color sequences, updating the buffer.
func (d *Decoder) Read(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	const maxBuf = 64 * 1024
	buf := make([]byte, maxBuf)
	scanner.Buffer(buf, maxBuf)
	scanner.Split(splitTwoBytes)
	for scanner.Scan() {
		tok := scanner.Bytes()
		chr := tok[0]
		atr := tok[1]
		if err := d.writeChar(chr, atr); err != nil {
			return err
		}
		if d.endOfRow() {
			d.writeRow()
			continue
		}
		d.column++
		if maxStop := d.maxRows > 0 && d.row > d.maxRows; maxStop {
			break
		}
	}
	// edge case, for handling tests or partially corrupted data dumps
	if d.maxRows == 0 && d.column != 1 {
		d.writeRow()
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "scan error:", err)
	}
	return nil
}

func splitTwoBytes(data []byte, atEOF bool) (int, []byte, error) {
	const advance = 2
	// return the two bytes as a color and character attribute token
	if len(data) >= advance {
		return advance, data[:2], nil
	}
	// if at EOF and there are leftover bytes, we discard them
	// as the token reader always expects two bytes.
	if atEOF {
		return 0, nil, bufio.ErrFinalToken
	}
	// request more data
	return 0, nil, nil
}

// decodeAttr returns the foreground and background
// colors that are an int between 0 and 15.
//
//nolint:mnd
func decodeAttr(b byte) (uint8, uint8) {
	fgLow := b & 0x07        // bits 0-2
	fgInt := (b >> 3) & 0x01 // bit 3
	bg := (b >> 4) & 0x07    // bits 4-6
	// blink := (b>>7)&0x01 == 1  // bit 7
	fg := fgLow | (fgInt << 3) // 0..15
	return fg, bg
}

func (d *Decoder) endOfRow() bool {
	n := d.column
	return n > 0 && n%d.columns == 0
}

//nolint:gosec
func (d *Decoder) writeChar(b, atr byte) error {
	const msg = "data is not a video binary dump"
	fg, bg := decodeAttr(atr)
	const lastColor = 15
	if fg > lastColor {
		return fmt.Errorf("%s %X foreground color, %d > 15: %w", msg, bg, bg, ErrAttribute)
	}
	if bg > lastColor {
		return fmt.Errorf("%s %X background color, %d > 15: %w", msg, bg, bg, ErrAttribute)
	}
	chr := html.EscapeString(string(d.charset.DecodeByte(b)))
	fgc := d.colors[fg].FG()
	bgc := d.colors[bg].BG()
	if d.Debug {
		// debug wraps every character within its own span element
		d.currentLine += template.HTML(`<span data-xy="` +
			fmt.Sprintf("%dx%d", d.row, d.column) +
			`" style="` + fgc + bgc + `">` + chr + `</span>`)
		return nil
	}
	// if the color attributes are identical to the colors used by the
	// previous character, then the character will be appended to the
	// span text content.
	// this should significantly reduce the size and node numbers of the
	// final HTML snippet
	if sameColors := d.column > 1 && d.currentAttr == atr; sameColors {
		d.currentLine += template.HTML(chr)
		return nil
	}
	if newline := d.column <= 1; newline {
		d.currentLine += template.HTML(`<span style="` + fgc + bgc + `">` + chr)
		d.currentAttr = atr
		return nil
	}
	// if colors have changed, we close the previous span element
	// and create a new element with the new color attributes.
	d.currentLine += template.HTML(`</span><span style="` + fgc + bgc + `">` + chr)
	d.currentAttr = atr
	return nil
}

func (d *Decoder) writeRow() {
	if !d.Debug {
		d.currentLine += `</span>`
	}
	d.buffer = append(d.buffer, d.currentLine+"\n")
	d.currentLine = ""
	d.row++
	d.column = 1
}

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
	"strconv"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

var (
	ErrAttribute = errors.New("attribute is not a 4-bit color value")
	ErrReader    = errors.New("reader is nil")
	//nolint:gochecknoglobals
	dumpTemplate = template.Must(template.New("dump").Parse(
		`{{define "T"}}<div>{{ . }}</div>{{end}}`))
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
	currentAttr byte
	fgStyles    [16]string // Pre-cached FG CSS strings
	bgStyles    [16]string // Pre-cached BG CSS strings
	lineBuilder *strings.Builder
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
		charset:     charset,
		columns:     width,
		column:      1,
		row:         1,
		maxRows:     0,
		lineBuilder: &strings.Builder{},
	}
	// Pre-allocate buffer for typical screen sizes (25-30 rows)
	bufferCapacity := 25
	if maxRows > 0 {
		d.maxRows = maxRows
		bufferCapacity = maxRows
	}
	d.buffer = make([]template.HTML, 0, bufferCapacity)
	switch pal { //nolint:exhaustive
	case RevisedCGA:
		d.colors = CGARevised()
	default:
		d.colors = CGA()
	}
	// Pre-cache CSS style strings
	for i := range 16 {
		d.fgStyles[i] = d.colors[i].FG()
		d.bgStyles[i] = d.colors[i].BG()
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

	d := NewDecoder(width, maxRows, pal, charset)
	if err := d.Read(r); err != nil {
		return nil, err
	}

	wr := new(bytes.Buffer)
	if err := d.Write(wr); err != nil {
		return nil, err
	}

	return wr, nil
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
//
// The return int64 is the number of bytes written.
func WriteTo(r io.Reader, w io.Writer) (int64, error) {
	const format = "buffer write to: %w"

	buf, err := Buffer(r, 0, 0, StandardCGA, nil)
	if err != nil {
		return 0, err
	}

	n, err := buf.WriteTo(w)
	if err != nil {
		return 0, fmt.Errorf(format, err)
	}
	return n, nil
}

// Write writes to w the full HTML fragment with outer div and inner lines joined with newlines.
func (d *Decoder) Write(wr io.Writer) error {
	const format = "write template execute: %w"
	if wr == nil {
		wr = io.Discard
	}
	total := 0
	for _, s := range d.buffer {
		total += len(s)
	}

	var sb strings.Builder
	sb.Grow(total)

	for _, s := range d.buffer {
		sb.WriteString(string(s))
	}

	//nolint:gosec
	data := template.HTML(sb.String())
	const name = "T"
	if err := dumpTemplate.ExecuteTemplate(wr, name, data); err != nil {
		return fmt.Errorf(format, err)
	}
	return nil
}

// Read reads each pair of bytes from r and interprets the color sequences, updating the buffer.
func (d *Decoder) Read(r io.Reader) error { //nolint:cyclop
	const format = "decoder read: %w"
	br := bufio.NewReader(r)
	const size = 2
	buf := make([]byte, size)

	for {
		_, err := io.ReadFull(br, buf)
		if err != nil {
			if err == io.EOF {
				break // normal end of stream
			}
			if errors.Is(err, io.ErrUnexpectedEOF) {
				break // corrupted tail, such as an odd number of bytes
			}
			return fmt.Errorf(format, err)
		}

		chr := buf[0]
		atr := buf[1]
		if err := d.writeChar(chr, atr); err != nil {
			return err
		}

		if d.endOfRow() {
			d.writeRow()
			continue
		}

		d.column++
		if d.maxRows > 0 && d.row > d.maxRows {
			break
		}
	}
	if d.maxRows == 0 && d.column != 1 {
		d.writeRow()
	}

	return nil
}

// decodeAttr returns the foreground and background
// colors that are an int between 0 and 15.
func decodeAttr(b byte) (uint8, uint8) {
	const colors = 0x07
	const intensity = 0x01
	const shiftInt = 3
	const shiftCol = 4
	fgLow := b & colors                  // bits 0-2
	fgInt := (b >> shiftInt) & intensity // bit 3
	bg := (b >> shiftCol) & colors       // bits 4-6
	// blink := (b>>7)&0x01 == 1  // bit 7
	fg := fgLow | (fgInt << shiftInt) // 0..15
	return fg, bg
}

func (d *Decoder) endOfRow() bool {
	return d.column >= d.columns
	// n := d.column
	// return n > 0 && n%d.columns == 0
}

func (d *Decoder) writeChar(b, atr byte) error {
	const format = "data is not a video binary dump %X %s color, %d > 15: %w"
	const lastColor = 15

	fg, bg := decodeAttr(atr)

	// Safety check (assumes fg/bg are unsigned integers so they can't be negative)
	if fg > lastColor {
		return fmt.Errorf(format, fg, "foreground", fg, ErrAttribute)
	}
	if bg > lastColor {
		return fmt.Errorf(format, bg, "background", bg, ErrAttribute)
	}

	chr := html.EscapeString(string(d.charset.DecodeByte(b)))
	fgc := d.fgStyles[fg]
	bgc := d.bgStyles[bg]

	if d.Debug {
		// debug wraps every character within its own span element
		d.lineBuilder.WriteString(`<span data-xy="`)
		d.lineBuilder.WriteString(strconv.Itoa(d.row))
		d.lineBuilder.WriteString(`x`)
		d.lineBuilder.WriteString(strconv.Itoa(d.column))
		d.lineBuilder.WriteString(`" style="`)
		d.lineBuilder.WriteString(fgc)
		d.lineBuilder.WriteString(bgc)
		d.lineBuilder.WriteString(`">`)
		d.lineBuilder.WriteString(chr)
		d.lineBuilder.WriteString(`</span>`)
		return nil
	}

	// If the color attributes are identical to the previous character,
	// append the character to the current span.
	if d.column > 1 && d.currentAttr == atr {
		d.lineBuilder.WriteString(chr)
		return nil
	}

	// Start of a new row: open the first span
	if d.column <= 1 {
		d.lineBuilder.WriteString(`<span style="`)
		d.lineBuilder.WriteString(fgc)
		d.lineBuilder.WriteString(bgc)
		d.lineBuilder.WriteString(`">`)
		d.lineBuilder.WriteString(chr)
		d.currentAttr = atr
		return nil
	}

	// Colors have changed mid-row: close the previous span and open a new one
	d.lineBuilder.WriteString(`</span><span style="`)
	d.lineBuilder.WriteString(fgc)
	d.lineBuilder.WriteString(bgc)
	d.lineBuilder.WriteString(`">`)
	d.lineBuilder.WriteString(chr)
	d.currentAttr = atr

	return nil
}

func (d *Decoder) writeRow() {
	if !d.Debug && d.column > 1 {
		d.lineBuilder.WriteString(`</span>`)
	}
	d.lineBuilder.WriteString("\n")
	//nolint:gosec
	d.buffer = append(d.buffer, template.HTML(d.lineBuilder.String()))
	d.lineBuilder.Reset()
	d.row++
	d.column = 1
}

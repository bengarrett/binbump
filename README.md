# BINbump

[![Go Reference](https://pkg.go.dev/badge/github.com/bengarrett/binbump.svg)](https://pkg.go.dev/github.com/bengarrett/binbump)
[![Go Report Card](https://goreportcard.com/badge/github.com/bengarrett/binbump)](https://goreportcard.com/report/github.com/bengarrett/binbump)

BINbump converts binary screen dumps of the IBM PC graphic and BIOS text mode characters, and CGA, EGA, and VGA colors 
into a HTML fragment for use in a template or webpage.

See the [reference documentation](https://pkg.go.dev/github.com/bengarrett/binbump) for usage, and examples, including changing the character sets and the color palette.

BINbump was created for and is in use on the website archive Defacto2, home to [thousands of ANSI and binary](https://defacto2.net/files/ansi) texts and artworks that are now rendered in HTML.

#### Quick usage

```go
package main

import (
	"log"
	"os"

	"github.com/bengarrett/binbump"
)

func main() {
	file, _ := os.Open("file.bin")
	defer file.Close()
	_, _ = binbump.WriteTo(file, os.Stdout)
}
```

#### HTML

BINbump will output a [`<div>`](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/div) "content division" element containing colors, styles, newlines, and text.
- The div element should be used within a [`<pre>`](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/pre) "preformatted text" element.
- Most ANSI text will want a custom monospaced font, [Cascadia Mono](https://github.com/microsoft/cascadia-code) handles all the [CodePage 437](https://en.wikipedia.org/wiki/Code_page_437) characters. 
- Or use the [IBM VGA font](https://int10h.org/oldschool-pc-fonts/fontlist/font?ibm_vga_8x16) for a more authentic recreation,
 either font will require a CSS [`@font-face`](https://developer.mozilla.org/en-US/docs/Web/CSS/@font-face) rule and [`font-family`](https://developer.mozilla.org/en-US/docs/Web/CSS/font-family) property.

```html
<html>
  <head>
    <title>Quick usage</title>
  </head>
  <style>
    @font-face {
      font-family: cascadia-mono;
      src: url(CascadiaMono.woff2) format("woff2");
    }
    pre {
      font-family: cascadia-mono, monospace, serif;
    }
  </style>
  <body>
    <pre><!--- binbump output ---><div><span style="color:#aaa;background-color:#000;">   </span><span style="color:#a50;background-color:#0a0;">HI‼︎</span><span style="color:#aaa;background-color:#000;">   </span></div>
    </pre>
  </body>
</html>
```

#### Not supported or known issues

- XBIN

#### Sauce metadata

BINbump doesn't parse any SAUCE metadata, however this can be done with a separate [bengarrett/sauce](https://github.com/bengarrett/sauce) package.

### Similar projects

- [Deark](https://github.com/jsummers/deark) is a utility that can output BIN to HTML or an image.
- [Ansilove](https://github.com/ansilove) is a collection of tools to convert BIN to images.
- [Ultimate Oldschool PC Font Pack](https://int10h.org/oldschool-pc-fonts/) offers various retro DOS and PC fonts.



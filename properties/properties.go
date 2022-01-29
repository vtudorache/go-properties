// Package properties manages persistent property tables. A table contains a
// hash of key-value pairs. It can be saved to or loaded from a stream. Each
// key and its corresponding value in the table is a string.
// A property table contains another property table as its "defaults".
// This secondary table is searched if the property key is not found in the
// primary table.
package properties

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// escapeRune writes into p the '\uxxxx' sequence representing the rune. If
// the rune is out of range or if no escaping is needed, writes the escape
// sequence of utf8.RuneError.
// If the rune is greater than 0xffff, writes the '\uxxxx' sequences of the
// two surrogates.
// It returns the number of bytes written.
func escapeRune(p []byte, r rune) int {
	if r > 0xffff {
		r1, r2 := utf16.EncodeRune(r)
		return escapeRune(p, r1) + escapeRune(p[6:], r2)
	}
	if 0x20 <= r && r <= 0x7e {
		return 0
	}
	if r < 0 {
		r = utf8.RuneError
	}
	p[0] = '\\'
	p[1] = 'u'
	for i := 5; i >= 2; i-- {
		b := byte(0x0f & r)
		if b > 9 {
			b += 'a' - 10
		} else {
			b += '0'
		}
		p[i] = b
		r >>= 4
	}
	return 6
}

// unescapeRune parses the first escape sequence in p. It recognizes the
// sequences '\t', '\n', '\f', '\r', '\uxxxx'. If a '\uxxxx' sequence holds
// a surrogate, a second '\uxxxx' sequence must be present, holding the
// next surrogate.
// It returns the rune and number of bytes parsed. If p doesn't start with
// an escape sequence, returns utf8.RuneError and 0.
func unescapeRune(p []byte) (rune, int) {
	n := len(p)
	if n < 1 || p[0] != '\\' {
		return utf8.RuneError, 0
	}
	r, size := utf8.DecodeRune(p[1:])
	if r == 't' {
		return '\t', 2
	}
	if r == 'n' {
		return '\n', 2
	}
	if r == 'f' {
		return '\f', 2
	}
	if r == 'r' {
		return '\r', 2
	}
	if r != 'u' {
		return r, size + 1
	}
	if n > 6 {
		n = 6
	}
	r = 0
	for i := 2; i < n; i++ {
		b := p[i]
		if '0' <= b && b <= '9' {
			b -= '0'
		} else if 'a' <= b && b <= 'f' {
			b -= 'a' - 10
		} else if 'A' <= b && b <= 'F' {
			b -= 'A' - 10
		} else {
			n = i
			break
		}
		r = (r << 4) | rune(b)
	}
	if n < 6 {
		r = utf8.RuneError
	}
	// here, n = 6 (the length of a '\uxxxx' sequence)
	if utf16.IsSurrogate(r) {
		q := r
		r, size = unescapeRune(p[6:])
		if size != 6 || !utf16.IsSurrogate(r) {
			return utf8.RuneError, 6
		}
		r = utf16.DecodeRune(q, r)
		n = 12
	}
	return r, n
}

func isDelimiter(r rune) bool {
	return (r == '=' || r == ':')
}

func isSpace(r rune) bool {
	return (r == '\t' || r == '\f' || r == ' ')
}

func isCmtPrefix(r rune) bool {
	return (r == '#' || r == '!')
}

func unescape(p []byte, split bool) (string, int) {
	var b strings.Builder
	n := 0
	for len(p) > 0 {
		r, size := unescapeRune(p)
		if size == 0 {
			r, size = utf8.DecodeRune(p)
			if split && (isSpace(r) || isDelimiter(r)) {
				p = p[size:]
				n += size
				for len(p) > 0 {
					r, size = utf8.DecodeRune(p)
					if !(isSpace(r) || isDelimiter(r)) {
						return b.String(), n
					}
					p = p[size:]
					n += size
				}
			}
		}
		b.WriteRune(r)
		p = p[size:]
		n += size
	}
	return b.String(), n
}

func loadBytes(r bufio.Reader) ([]byte, error) {
	var b []byte
	done := false
	for !done {
		x, e := r.ReadByte()
		if e != nil {
			return b, e
		}
		for x == '\t' || x == '\f' || x == ' ' {
			x, e = r.ReadByte()
			if e != nil {
				return b, e
			}
		}
		if (x == '#' || x == '!') && len(b) == 0 {
			done = true
		}
		esc := false
		for x != '\n' && x != '\r' {
			if x == '\\' {
				esc = !esc
			} else {
				esc = false
			}
			b = append(b, x)
			x, e = r.ReadByte()
			if e != nil {
				return b, e
			}
		}
		if x == '\r' {
			x, e = r.ReadByte()
			if e != nil {
				return b, e
			}
		}
		if x != '\n' {
			e = r.UnreadByte()
			if e != nil {
				return b, e
			}
		}
		if !done {
			if esc {
				b = b[:len(b)-1]
			} else {
				done = true
			}
		}
	}
	return b, nil
}

// Table represents a property table. It contains a hash of key-value pairs.
// It also contains a secondary property table as its "defaults". The
// secondary table is searched if the property key was not found in the
// primary table.
type Table struct {
	data     map[string]string
	defaults *Table
}

// Load reads a property table (key and value pairs) from the reader in a
// line-oriented format. Properties are processed in terms of lines. There
// are two kinds of lines, partial lines and full lines. A partial line is a
// line of characters ending either by the standard terminators ('\n', '\r'
// or '\r\n') or by the end-of-file. A partial line may be blank, or a
// comment line, or it may hold all or some of a key-value pair. A full line
// holds all the data of a key-value pair. It may spread across several
// adjacent partial lines by escaping the line terminators with the backslash
// character '\'. Comment lines can't spread. A partial line holding a
// comment must have its own comment prefix.
// Lines are read from the input until the end-of-file is reached.
// A partial line containing only white space characters is considered empty
// and is ignored. A comment line has an ASCII '#' or '!' as its first
// non-space character. Comment lines are ignored and do not encode key-value
// data.
// The characters ' ' ('\u0020'), '\t' ('\u0009'), and '\f' ('\u000C')
// are considered space.
// If a full line spreads on several partial lines, the backslash escaping
// the line terminator sequence, the line terminator sequence, and any space
// character at the start of the following line have no effect on the key or
// value. It is not enough to have a backslash preceding a line terminator
// sequence to escape the terminator. There must be an odd number of
// contiguous backslashes for the line terminator to be escaped. Since the
// input is processed from left to right, a number of 2n contiguous
// backslashes encodes n backslashes after unescaping.
// The key contains all of the characters in the line starting with the first
// non-space character and up to, but not including, the first unescaped '=',
// ':', or white space character other than a line terminator. All of these
// key delimiters may be included in the key by escaping them with a
// backslash character. For example,
// ```
// \=\:\=
// ```
// would be the key "=:=". Line terminators can be included using '\r' and
// '\n' escape sequences. Any space character after the key is skipped. If
// the first non-white space character after the key is '=' or ':', then it
// is ignored and any space characters after it are also skipped. All
// remaining characters on the line become part of the associated value. If
// there are no remaining characters, the value is the empty string "".
// As an example, each of the following lines specifies the key "Go" and the
// associated value "The Best Language":
// ```
// Go = The Best Language
//     Go:The Best Language
// Go                    :The Best Language
// ```
// As another example, the following lines specify a single property:
// ```
// languages                       Assembly, Lisp, Pascal, \
//                                 BASIC, C, \
//                                 Perl, Tcl, Lua, Java, Python, \
//                                 C#, Go
// ```
// The key is "languages" and the associated value is:
// "Assembly, Lisp, Pascal, BASIC, C, Perl, Tcl, Lua, Java, Python, C#, Go".
// Note that a space appears before each '\' so that a space will appear in
// the final result; the '\', the line terminator, and the leading spaces
// on the continuation line are discarded and not replaced by other
// characters.
// Octal escapes are not recognized. The character sequence '\b' does not
// represent a backspace character. A backslash character before a non-valid
// escape character is not an error, the backslash is silently dropped.
// Escapes are not necessary for single and double quotes, however, by the
// rule above, single and double quote characters preceded by a backslash
// yield single and double quote characters, respectively. Only a single 'u'
// character is allowed in a Unicode escape sequence. Unicode runes above
// 0xffff should be stored as two consecutive '\uxxxx' sequeces encoding the
// surrogates.
// Returns the number of key-value pairs loaded and any error encountered.
func (p *Table) Load(r io.Reader) (int, error) {
	var reader = bufio.NewReader(r)
	count := 0
	done := false
	for !done {
		b, e := loadBytes(*reader)
		if len(b) > 0 && b[0] != '#' && b[0] != '!' {
			key, i := unescape(b, true)
			value, _ := unescape(b[i:], false)
			p.data[key] = value
			count += 1
		}
		if e != nil {
			if e != io.EOF {
				return count, e
			}
			done = true
		}
	}
	return count, nil
}

// LoadString loads a property table using the given string as input. It
// returns the number of key-value pairs loaded and any error encountered.
func (p *Table) LoadString(s string) (int, error) {
	r := strings.NewReader(s)
	return p.Load(r)
}

func escape(key, value string, ascii bool) []byte {
	var b bytes.Buffer
	var buffer [12]byte
	var r rune
	for _, r = range key {
		size := 0
		if ascii {
			size = escapeRune(buffer[:], r)
		}
		if size == 0 {
			if r == '\n' {
				b.WriteString("\\n")
				continue
			}
			if r == '\r' {
				b.WriteString("\\r")
				continue
			}
			if isSpace(r) || isDelimiter(r) || isCmtPrefix(r) {
				b.WriteByte('\\')
			}
			size = utf8.EncodeRune(buffer[:], r)
		}
		b.Write(buffer[:size])
	}
	b.WriteRune('=')
	r, _ = utf8.DecodeRuneInString(value)
	if isSpace(r) || isDelimiter(r) {
		b.WriteByte('\\')
	}
	for _, r = range value {
		size := 0
		if ascii {
			size = escapeRune(buffer[:], r)
		}
		if size == 0 {
			if r == '\n' {
				b.WriteString("\\n")
				continue
			}
			if r == '\r' {
				b.WriteString("\\r")
				continue
			}
			if isCmtPrefix(r) {
				b.WriteByte('\\')
			}
			size = utf8.EncodeRune(buffer[:], r)
		}
		b.Write(buffer[:size])
	}
	return b.Bytes()
}

func escapeText(text string, ascii bool) []byte {
	var b bytes.Buffer
	var buffer [12]byte
	last := rune('\n')
	for _, r := range text {
		if r == '\n' || r == '\r' {
			b.WriteRune(r)
			last = r
			continue
		}
		if (last == '\n' || last == '\r') && !isCmtPrefix(r) {
			b.WriteByte('#')
		}
		size := 0
		if ascii {
			size = escapeRune(buffer[:], r)
		}
		if size == 0 {
			size = utf8.EncodeRune(buffer[:], r)
		}
		b.Write(buffer[:size])
		last = r
	}
	return b.Bytes()
}

// Store writes this property table (key and element pairs) to w in a format
// suitable for using the Load method. The properties in the defaults table
// (if any) are not written out by this method.
// If ascii is true, then any rune lesser than 0x20 or greater than 0x7e is
// converted to its '\uxxxx'  escape sequence(s).
// Every key-value pair in the table is written out, one per line. For each
// entry, the key is written, then an ASCII '=', then the associated value.
// For the key, all space characters are written with a preceding '\'
// character. For the value, leading space characters, but not embedded or
// trailing space characters, are written with a preceding '\' character.
// The key and value characters '#', '!', '=', and ':' are written with a
// preceding '\' to ensure that they are properly loaded.
// The function returns the number of key-value pairs written and any error
// encountered.
func (p *Table) Store(w io.Writer, ascii bool) (int, error) {
	count := 0
	eol := []byte("\n")
	for key, value := range p.data {
		if _, e := w.Write(escape(key, value, ascii)); e != nil {
			return count, e
		}
		if _, e := w.Write(eol); e != nil {
			return count, e
		}
		count += 1
	}
	return count, nil
}

// Save writes this property table (key and element pairs) to w in a format
// suitable for using the Load method. The properties in the defaults table
// (if any) are not written out by this method.
// If ascii is true, then any rune lesser than 0x20 or greater than 0x7e is
// converted to its '\uxxxx'  escape sequence(s).
// If comments is not empty, then an ASCII '#' character, the comments
// string, and a line separator are first written to w. Any set of line
// terminators is replaced by a line separator and if the next character
// in comments is not '#' or '!', then an ASCII '#' is written out after that
// line separator.
// Then every entry in the table is written out, one per line. For each
// entry, the key is written, then an ASCII '=', then the associated value.
// For the key, all space characters are written with a preceding '\'
// character. For the value, leading space characters, but not embedded or
// trailing space characters, are written with a preceding '\' character.
// The key and value characters '#', '!', '=', and ':' are written with a
// preceding '\' to ensure that they are properly loaded.
// The function returns the number of key-value pairs written and any error
// encountered.
func (p *Table) Save(w io.Writer, comments string, ascii bool) (int, error) {
	eol := []byte("\n")
	if _, e := w.Write(escapeText(comments, ascii)); e != nil {
		return 0, e
	}
	if _, e := w.Write(eol); e != nil {
		return 0, e
	}
	return p.Store(w, ascii)
}

// SaveString returns the text form of the property table and any error
// encountered. The ascii parameter has the same meaning as for the Save
// function above.
func (p *Table) SaveString(comments string, ascii bool) (string, error) {
	var b strings.Builder
	_, e := p.Save(&b, comments, ascii)
	return b.String(), e
}

// String returns a text representation (as UTF-8) of the property table (not
// including the key-value pairs of the secondary table). The text can be
// then reused by LoadString.
func (p *Table) String() string {
	var b strings.Builder
	eol := []byte("\n")
	for key, value := range p.data {
		b.Write(escape(key, value, false))
		b.Write(eol)
	}
	return b.String()
}

// NewTableWith creates and initializes a new property table using defaults
// for the secondary table.
func NewTableWith(defaults *Table) *Table {
	return &Table{
		map[string]string{},
		defaults,
	}
}

// NewTable creates and initializes a new property table with no secondary
// table.
func NewTable() *Table {
	return NewTableWith(nil)
}

// Lookup searches the value associated with key. If key isn't present in the
// primary table, the function searches the secondary table. It returns the
// value (or the empty string) and a boolean indicating whether the value was
// found or not.
func (p *Table) Lookup(key string) (string, bool) {
	if value, found := p.data[key]; found {
		return value, true
	}
	if p.defaults != nil {
		if value, found := p.defaults.Lookup(key); found {
			return value, true
		}
	}
	return "", false
}

// Get returns the value associated with the string key. If key isn't present
// in the primary table, it searches the secondary table. If the key isn't
// found, returns the empty string.
func (p *Table) Get(key string) string {
	value, _ := p.Lookup(key)
	return value
}

// Set associates key with value in the property table. If key is already
// present in the table, the associated value is replaced.
func (p *Table) Set(key string, value string) {
	p.data[key] = value
}

// Delete removes the key and the associated value from the property table.
// If the key isn't present, calling this function does nothing.
func (p *Table) Delete(key string) {
	delete(p.data, key)
}

// Clear deletes all the key-value pairs in the primary table. It doesn't
// delete the pairs in the secondary table.
func (p *Table) Clear() {
	p.data = make(map[string]string)
}

// ClearAll deletes all the key-value pairs in the primary and the secondary
// property tables.
func (p *Table) ClearAll() {
	p.Clear()
	if p.defaults != nil {
		p.defaults.ClearAll()
	}
}

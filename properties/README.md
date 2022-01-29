# Overview

Package properties manages persistent property tables. A table contains a 
hash of key-value pairs. It can be saved to or loaded from a stream. Each 
key and its corresponding value in the table is a string.  
A property table contains another property table as its "defaults". 
This secondary table is searched if the property key is not found in the
primary table.

# Index

[type Table](#type-table)  
[func NewTable() *Table](#func-newtable)  
[func NewTableWith(defaults *Table) *Table](#func-newtablewith)  
[func (p *Table) Clear()](#func-p-table-clear)  
[func (p *Table) ClearAll()](#func-p-table-clearall)  
[func (p *Table) Delete(key string)](#func-p-table-delete)  
[func (p *Table) Get(key string) string](#func-p-table-get)  
[func (p *Table) Load(r io.Reader) (int, error)](#func-p-table-load)  
[func (p *Table) LoadString(s string) (int, error)](#func-p-table-load-string)  
[func (p *Table) Lookup(key string) (string, bool)](#func-p-table-lookup)  
[func (p *Table) Save(w io.Writer, comments string, ascii bool) (int, error)](#func-p-table-save)  
[func (p *Table) SaveString(comments string, ascii bool) (string, error)](#func-p-table-savestring)  
[func (p *Table) Set(key string, value string)](#func-p-table-set)  
[func (p *Table) String() string](#func-p-table-string)  
[func (p *Table) Store(w io.Writer, ascii bool) (int, error)](#func-p-table-store)  

## type Table
```
type Table struct {
    // contains filtered or unexported fields
}
```
Table represents a property table. It contains a hash of key-value pairs. 
It also contains a secondary property table as its "defaults". The 
secondary table is searched if the property key was not found in the 
primary table.

## func NewTable
```
func NewTable() *Table
```  
NewTable creates and initializes a new property table with no secondary 
table.

## func NewTableWith
```
func NewTableWith(defaults *Table) *Table  
```
NewTableWith creates and initializes a new property table using defaults for 
the secondary table.

## func (p *Table) Clear  
```
func (p *Table) Clear()
```
Clear deletes all the key-value pairs in the primary table. It doesn't delete 
the pairs in the secondary table.

## func (p *Table) ClearAll  
```
func (p *Table) ClearAll()
```
ClearAll deletes all the key-value pairs in the primary and the secondary 
property tables.

## func (p *Table) Delete
```
func (p *Table) Delete(key string)
```
Delete removes the key and the associated value from the property table. If the
key isn't present, calling this function does nothing.

## func (p *Table) Get
```
func (p *Table) Get(key string) string  
```
Get returns the value associated with the string key. If key isn't present in
the primary table, it searches the secondary table. If the key isn't found, 
returns the empty string.

## func (p *Table) Load
```
func (p *Table) Load(r io.Reader) (int, error)
```
Load reads a property table (key and value pairs) from the reader in a 
line-oriented format. Properties are processed in terms of lines. There are two 
kinds of lines, partial lines and full lines. A partial line is a line of 
characters ending either by the standard terminators ('\\n', '\\r' or '\\r\\n') 
or by the end-of-file. A partial line may be blank, or a comment line, or it 
may hold all or some of a key-value pair. A full line holds all the data of 
a key-value pair. It may spread across several adjacent partial lines by 
escaping the line terminators with the backslash character '\\'. Comment lines 
can't spread. A partial line holding a comment must have its own comment 
prefix.  
Lines are read from the input until the end-of-file is reached.
A partial line containing only white space characters is considered empty and 
is ignored. A comment line has an ASCII '#' or '!' as its first non-space 
character. Comment lines are ignored and do not encode key-value data.  
The characters ' ' ('\\u0020'), '\\t' ('\\u0009'), and '\\f' ('\\u000C') are 
considered space.  
If a full line spreads on several partial lines, the backslash escaping the 
line terminator sequence, the line terminator sequence, and any space character 
at the start of the following line have no effect on the key or value.
It is not enough to have a backslash preceding a line terminator sequence to 
escape the terminator. There must be an odd number of contiguous backslashes 
for the line terminator to be escaped. Since the input is processed from left 
to right, a number of 2n contiguous backslashes encodes n backslashes after 
unescaping.  
The key contains all of the characters in the line starting with the first 
non-space character and up to, but not including, the first unescaped '=', ':', 
or white space character other than a line terminator. All of these
key delimiters may be included in the key by escaping them with a backslash
character. For example,
```
\=\:\=
```
would be the key "=:=". Line terminators can be included using '\\r' and '\\n' 
escape sequences. Any space character after the key is skipped. If the first 
non-white space character after the key is '=' or ':', then it is ignored and 
any space characters after it are also skipped. All remaining characters on 
the line become part of the associated value. If there are no remaining 
characters, the value is the empty string "".  
As an example, each of the following lines specifies the key "Go" and the
associated value "The Best Language":
```
Go = The Best Language
    Go:The Best Language
Go                    :The Best Language
```
As another example, the following lines specify a single property:
```
languages                       Assembly, Lisp, Pascal, \
                                BASIC, C, \
                                Perl, Tcl, Lua, Java, Python, \
                                C#, Go
```
The key is "languages" and the associated value is:  
"Assembly, Lisp, Pascal, BASIC, C, Perl, Tcl, Lua, Java, Python, C#, Go".  
Note that a space appears before each '\\' so that a space will appear in the
final result; the '\\', the line terminator, and the leading spaces on the
continuation line are discarded and not replaced by other characters.  
Octal escapes are not recognized. The character sequence '\\b' does not
represent a backspace character. A backslash character before a non-valid
escape character is not an error, the backslash is silently dropped.
Escapes are not necessary for single and double quotes, however, by the
rule above, single and double quote characters preceded by a backslash
yield single and double quote characters, respectively. Only a single 'u'
character is allowed in a Unicode escape sequence. Unicode runes above
0xffff should be stored as two consecutive '\\uxxxx' sequeces encoding the
surrogates.  
Returns the number of key-value pairs loaded and any error encountered.

## func (p *Table) LoadString  
```
func (p *Table) LoadString(s string) (int, error)  
```
LoadString loads a property table using the given string as input. It returns
the number of key-value pairs loaded and any error encountered.

## func (p *Table) Lookup  
```
func (p *Table) Lookup(key string) (string, bool)
```
Lookup searches the value associated with key. If key isn't present in the
primary table, the function searches the secondary table. It returns the
value (or the empty string) and a boolean indicating whether the value was
found or not.

## func (p *Table) Save  
```
func (p *Table) Save(w io.Writer, comments string, ascii bool) (int, error)  
```
Save writes this property table (key and element pairs) to w in a format 
suitable for using the Load method. The properties in the defaults table (if 
any) are not written out by this method.  
If ascii is true, then any rune lesser than 0x20 or greater than 0x7e is 
converted to its '\\uxxxx'  escape sequence(s).  
If comments is not empty, then an ASCII '#' character, the comments string, and 
a line separator are first written to w. Any set of line terminators is 
replaced by a line separator and if the next character in comments is not '#' 
or '!', then an ASCII '#' is written out after that line separator.  
Then every entry in the table is written out, one per line. For each entry, the 
key is written, then an ASCII '=', then the associated value. For the key, all 
space characters are written with a preceding '\\' character. For the value, 
leading space characters, but not embedded or trailing space characters, are 
written with a preceding '\\' character. The key and value characters '#', '!', 
'=', and ':' are written with a preceding '\\' to ensure that they are properly 
loaded.  
The function returns the number of key-value pairs written and any error 
encountered.

## func (p *Table) SaveString
```  
func (p *Table) SaveString(comments string, ascii bool) (string, error)  
```
SaveString returns the text form of the property table and any error 
encountered. It uses the [Save](#func-p-table-save) function. The ascii 
parameter has the same meaning.

## func (p *Table) Set  
```
func (p *Table) Set(key string, value string)  
```
Set associates key with value in the property table. If key is already
present in the table, the associated value is replaced.

## func (p *Table) String  
```
func (p *Table) String() string
```  
String returns a text representation (as UTF-8) of the property table (not
including the key-value pairs of the secondary table). The text can be then
reused by LoadString.

## func (p *Table) Store
```  
func (p *Table) Store(w io.Writer, ascii bool) (int, error)
```
Store writes this property table (key and element pairs) to w in a format 
suitable for using the Load method. The properties in the defaults table (if 
any) are not written out by this method.  
If ascii is true, then any rune lesser than 0x20 or greater than 0x7e is 
converted to its '\\uxxxx'  escape sequence(s).  
Every key-value pair in the table is written out, one per line. For each entry, 
the key is written, then an ASCII '=', then the associated value. For the key, 
all space characters are written with a preceding '\\' character. For the 
value, leading space characters, but not embedded or trailing space characters, 
are written with a preceding '\\' character. The key and value characters '#', 
'!', '=', and ':' are written with a preceding '\\' to ensure that they are 
properly loaded.  
The function returns the number of key-value pairs written and any error 
encountered.


package properties

import (
	"testing"
)

func TestLoadString(t *testing.T) {
	p := NewTable()
	p.LoadString("firstKey=firstValue")
	p.LoadString("second\\ key = second value")
	p.LoadString("third\\ key third \\\n  \textended value")
	p.LoadString("fourth\\ key\\ : \\ fourth value\n")
	p.LoadString("fifth\\ key = fifth value with \\u20ac")
	if p.Get("firstKey") != "firstValue" {
		t.Error(`p.Get("firstKey") != "firstValue"`)
	}
	if p.Get("second key") != "second value" {
		t.Error(`p.Get("second key") != "second value"`)
	}
	if p.Get("third key") != "third extended value" {
		t.Error(`p.Get("third key") != "third extended value"`)
	}
	if p.Get("fourth key ") != " fourth value" {
		t.Error(`p.Get("fourth key ") != " fourth value"`)
	}
	if p.Get("fifth key") != "fifth value with €" {
		t.Error(`p.Get("fifth key") != "fifth value with €"`)
	}
}

func TestSaveString(t *testing.T) {
	var p *Table
	var s string
	p = NewTable()
	p.Set("firstKey", "firstValue")
	s, _ = p.SaveString("The first\r\nproperties entry", false)
	if s != "#The first\r\n#properties entry\nfirstKey=firstValue\n" {
		t.Error("SaveString() returned ", s)
	}
	p.Clear()
	p.Set("second key", "second value")
	s, _ = p.SaveString("!The second property", false)
	if s != "!The second property\nsecond\\ key=second value\n" {
		t.Error("SaveString() returned ", s)
	}
	p.Clear()
	p.Set("third #key", "third !value")
	s, _ = p.SaveString("The third property", false)
	if s != "#The third property\nthird\\ \\#key=third \\!value\n" {
		t.Error("SaveString() returned ", s)
	}
	p.Clear()
	p.Set("fourth \n#key", "fourth !value")
	s, _ = p.SaveString("The fourth property", false)
	if s != "#The fourth property\nfourth\\ \\n\\#key=fourth \\!value\n" {
		t.Error("SaveString() returned ", s)
	}
	p.Clear()
	p.Set("fifth key", "fifth value with €")
	s, _ = p.SaveString("The fifth property", true)
	if s != "#The fifth property\nfifth\\ key=fifth value with \\u20ac\n" {
		t.Error("SaveString() returned ", s)
	}
	p.Clear()
	p.Set("sixth key", "sixth value with 😀 objects")
	s, _ = p.SaveString("The sixth property", true)
	if s != "#The sixth property\nsixth\\ key=sixth value with \\ud83d\\ude00 objects\n" {
		t.Error("SaveString() returned ", s)
	}
}

func TestDefaults(t *testing.T) {
	var p *Table
	var s string
	p = NewTable()
	p.LoadString("firstKey=firstValue")
	p.LoadString("second\\ key = second value")
	p.LoadString("third\\ key third \\\n  \textended value")
	p.LoadString("fourth\\ key\\ : \\ fourth value\n")
	p = NewTableWith(p)
	if p.Get("firstKey") != "firstValue" {
		t.Error(`p.Get("firstKey") != "firstValue"`)
	}
	if p.Get("second key") != "second value" {
		t.Error(`p.Get("second key") != "second value"`)
	}
	if p.Get("third key") != "third extended value" {
		t.Error(`p.Get("third key") != "third extended value"`)
	}
	if p.Get("fourth key ") != " fourth value" {
		t.Error(`p.Get("fourth key ") != " fourth value"`)
	}
	s, _ = p.SaveString("Table with defaults", false)
	if s != "#Table with defaults\n" {
		t.Error("SaveString() returned ", s)
	}
	p.Set("fourth key", "a new fourth value")
	s, _ = p.SaveString("Table with defaults", false)
	if s != "#Table with defaults\nfourth\\ key=a new fourth value\n" {
		t.Error("SaveString() returned ", s)
	}
}

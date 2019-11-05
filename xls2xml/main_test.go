package main

import (
	"testing"
	"time"
)

var line = map[string]string{
	"id":   "1",
	"name": "2",
}

func TestFunctionDict(t *testing.T) {
	// x := FunctionDict["fixed"]
	// arg := "aaa"
	// exp := "aaa"
	// res, err := x(arg, line)
	// if err != nil {
	// 	t.Errorf("%v", err)
	// }
	// if exp != res {
	// 	t.Errorf("Fixed function wrong: fixed(\"%s\") = %s", arg, exp)
	// }
}

func TestRemoveSpaces(t *testing.T) {
	tables := []struct {
		arg string
		exp string
	}{
		{"", ""},
		{"  ", ""},
		{" aa aa  aa   aa ", "aa_aa_aa_aa"},
	}

	for _, table := range tables {
		res := RemoveSpaces(table.arg)
		if res != table.exp {
			t.Errorf("RemoveSpaces(\"%s\") = [%s], expected [%s]", table.arg, res, table.exp)
		}
	}
}

func TestRemoveQuotes(t *testing.T) {
	tables := []struct {
		arg string
		exp string
	}{
		{"", ""},
		{"'", ""},
		{"\"", ""},
		{"  ", "  "},
		{"'aa'aa'", "aaaa"},
		{"'aa\"'\"a\"a'", "aaaa"},
	}

	for _, table := range tables {
		res := RemoveQuotes(table.arg)
		if res != table.exp {
			t.Errorf("RemoveQuotes(\"%s\") = [%s], expected [%s]", table.arg, res, table.exp)
		}
	}
}

func TestRemoveAccents(t *testing.T) {
	tables := []struct {
		arg string
		exp string
	}{
		{"", ""},
		{"ć", "c"},
		{"cçc", "ccc"},
		{"!@#$%¨&*()", "!@#$%¨&*()"},
		{"abcdefghijklmnopqrstuvwxyzáéíóúçãõâêîôûàèìòù", "abcdefghijklmnopqrstuvwxyzaeioucaoaeiouaeiou"},
		{"Titulo Original", "Titulo Original"},
	}

	for _, table := range tables {
		res, err := RemoveAccents(table.arg)
		if err != nil {
			t.Errorf("%v", err)
			continue
		}
		if res != table.exp {
			t.Errorf("RemoveAccents(\"%s\") = [%s], expected [%s]", table.arg, res, table.exp)
		}
	}
}

func TestFormatMoney(t *testing.T) {
	tables := []struct {
		arg string
		exp string
	}{
		{"", ""},
		{"0", "0.00"},
		{"444", "444.00"},
		{"123.4567", "123.46"},
		{"-123.4567", "-123.46"},
	}

	for _, table := range tables {
		res, err := formatMoney(table.arg)
		if err != nil {
			t.Errorf("%v", err)
			continue
		}
		if res != table.exp {
			t.Errorf("formatMoney(\"%s\") = [%s], expected [%s]", table.arg, res, table.exp)
		}
	}
}

func TestFormatTimestamp(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2005-11-29T22:08:41+00:00")
	tables := []struct {
		arg time.Time
		exp string
	}{
		{t1, "20051129220841"},
	}

	for _, table := range tables {
		res := formatTimestamp(table.arg)
		if res != table.exp {
			t.Errorf("formatTimeStamp(\"%s\") = [%s], expected [%s]", table.arg, res, table.exp)
		}
	}
}

func TestFormatDate(t *testing.T) {
	t1, _ := time.Parse(time.RFC3339, "2005-11-29T22:08:41+00:00")
	tables := []struct {
		arg time.Time
		exp string
	}{
		{t1, "2005-11-29"},
	}

	for _, table := range tables {
		res := formatDate(table.arg)
		if res != table.exp {
			t.Errorf("formatDate(\"%s\") = [%s], expected [%s]", table.arg, res, table.exp)
		}
	}
}

func TestCondition(t *testing.T) {
	line := map[string]string{"a": "1", "b": "22", "c": "1"}
	tables := []struct {
		arg string
		exp bool
	}{
		{"a=='1'", true},
		{"b=='22'", true},
		{"a==b", false},
		{"a==c", true},
		{"a==c && a!=b", true},
		{"a==c || a==b", true},
		{"a != c || (a==c && a!=b)", true},
		{"a == c && (a!=c || a==b)", false},
	}

	for _, table := range tables {
		res, err := evalCondition(table.arg, line)
		if err != nil {
			t.Error(err)
		}
		if res != table.exp {
			t.Errorf("evalCondition(\"%s\") = [%v], expected [%v]", table.arg, res, table.exp)
		}
	}
}

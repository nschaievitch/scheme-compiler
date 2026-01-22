package main

import (
	"reflect"
	"testing"
)

func tryParse(t *testing.T, source string, expected any) {
	prog, err := SchemeParse(source)

	if err != nil {
		t.Errorf("Unexpected error while parsing string %s: expected %v, got the following error %e", source, expected, err)
	}

	if !reflect.DeepEqual(prog, expected) {
		t.Errorf("Unexpected parsing result. Expected %v, got %v instead", expected, prog)
	}
}

func TestParsesNumber(t *testing.T) {
	tryParse(t, "42", Uint{42})
}

func TestParsesNumberWithWhitespace(t *testing.T) {
	tryParse(t, "   \n42 \n \t", Uint{42})
}

func TestParsesBoolean(t *testing.T) {
	tryParse(t, "#t", Boolean{true})
	tryParse(t, "#f", Boolean{false})
}
func TestParsesBooleanWithWhitespace(t *testing.T) {
	tryParse(t, "\n\n\t#t\n", Boolean{true})
}

func TestParsesChar(t *testing.T) {
	tryParse(t, "#\\a", Char{'a'})
	tryParse(t, "#\\-", Char{'-'})
	tryParse(t, "#\\(", Char{'('})
	tryParse(t, "#\\#", Char{'#'})
	tryParse(t, "#\\\\", Char{'\\'})
}

func TestParsesCharWithWhitespace(t *testing.T) {
	tryParse(t, "    \t\t\r\n#\\a\r\n", Char{'a'})
}

func TestErrorsOnInvalidPoundLiteral(t *testing.T) {
	prog, err := SchemeParse("#")

	if err == nil {
		t.Errorf("Expected error parsing invalid litaral, got %v instead", prog)
	}
}

func TestErrorsOnInvalidPoundCharacter(t *testing.T) {
	prog, err := SchemeParse("#d")

	if err == nil {
		t.Errorf("Expected error parsing invalid litaral, got %v instead", prog)
	}
}

func TestErrorsOnIncompleteCharacter(t *testing.T) {
	prog, err := SchemeParse("#\\")

	if err == nil {
		t.Errorf("Expected error parsing invalid litaral, got %v instead", prog)
	}
}

func TestParsesIdentifier(t *testing.T) {
	tryParse(t, "+", Identifier{"+"})
	tryParse(t, "name", Identifier{"name"})
	tryParse(t, "integer->char", Identifier{"integer->char"})
	tryParse(t, "\n\n\t integer->char", Identifier{"integer->char"})
}

func TestParsesSExp(t *testing.T) {
	tryParse(t, "(1)", SExp{[]Node{Uint{1}}})
	tryParse(t, "(integer->char 2)", SExp{[]Node{Identifier{"integer->char"}, Uint{2}}})
	tryParse(t, "(+ 1 2)", SExp{[]Node{Identifier{"+"}, Uint{1}, Uint{2}}})
}

func TestParsesNestedSExp(t *testing.T) {
	tryParse(t, "(+ 1 (- 1 2))", SExp{[]Node{Identifier{"+"}, Uint{1}, SExp{[]Node{Identifier{"-"}, Uint{1}, Uint{2}}}}})
	tryParse(t, "(+ (- 1 2) 1)", SExp{[]Node{Identifier{"+"}, SExp{[]Node{Identifier{"-"}, Uint{1}, Uint{2}}}, Uint{1}}})
}

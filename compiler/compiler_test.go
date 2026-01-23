package main

import (
	"reflect"
	"testing"
)

func tryCompile(t *testing.T, source string, expected []Word) {
	c := Compiler{}

	program, err := SchemeParse(source)

	if err != nil {
		t.Fatalf("Unexpected error while parsing string %s, got the following error %e", source, err)
	}

	env := make(Environment)

	err = c.CompileFunction(program, &env)

	if err != nil {
		t.Fatalf("Unexpected error while compiling string %s: expected %v, got the following error %e", source, expected, err)
	}

	if !reflect.DeepEqual(expected, c.Code) {
		t.Errorf("Unexpected compile output while compiling string %s. Expected %v, got %v instead", source, expected, c.Code)
	}
}

func TestCompileNumber(t *testing.T) {
	tryCompile(t, "42", []Word{LOAD64, 42 << 2, RETURN})
	tryCompile(t, "0", []Word{LOAD64, 0, RETURN})
}

func TestCompileChar(t *testing.T) {
	tryCompile(t, "#\\a", []Word{LOAD_CHAR, Word((uint64('a') << 8) | 0b00001111), RETURN})
	tryCompile(t, "#\\.", []Word{LOAD_CHAR, Word((uint64('.') << 8) | 0b00001111), RETURN})
}

func TestCompileBoolean(t *testing.T) {
	tryCompile(t, "#f", []Word{LOAD_BOOL, Word(0b0011111), RETURN})
	tryCompile(t, "#t", []Word{LOAD_BOOL, Word(0b10011111), RETURN})
}

func TestCompilesUnaryExpression(t *testing.T) {
	tryCompile(t, "(add1 1)", []Word{LOAD64, Word(1 << 2), ADD1, RETURN})
	tryCompile(t, "(char->integer #\\a)", []Word{LOAD_CHAR, Word(uint64('a')<<8 | 0b00001111), CHAR_TO_INTEGER, RETURN})
}

func TestCompilesBinaryExpression(t *testing.T) {
	tryCompile(t, "(+ 1 2)", []Word{LOAD64, 2 << 2, LOAD64, 1 << 2, PLUS, RETURN})
	tryCompile(t, "(= #t #f)", []Word{LOAD_BOOL, Word(0b0011111), LOAD_BOOL, Word(0b10011111), EQUALP, RETURN})
}

func TestCompilesNestedExpression(t *testing.T) {
	tryCompile(t, "(+ 1 (+ 1 2))", []Word{LOAD64, 2 << 2, LOAD64, 1 << 2, PLUS, LOAD64, 1 << 2, PLUS, RETURN})
	tryCompile(t, "(= (- 4 1) (+ 2 1))", []Word{LOAD64, 1 << 2, LOAD64, 2 << 2, PLUS, LOAD64, 1 << 2, LOAD64, 4 << 2, MINUS, EQUALP, RETURN})
}

func TestCompileLet(t *testing.T) {
	tryCompile(t, "(let (a 1) a)", []Word{LOAD64, 1 << 2, LOAD_VAR, 0, RETURN})
	tryCompile(t, "(let (a 1) (b 2) (+ a b))", []Word{LOAD64, 1 << 2, LOAD64, 2 << 2, LOAD_VAR, 1, LOAD_VAR, 0, PLUS, RETURN})
	tryCompile(t, "(let (a (+ 1 2)) (b 10) (- b a))", []Word{LOAD64, 2 << 2, LOAD64, 1 << 2, PLUS, LOAD64, 10 << 2, LOAD_VAR, 0, LOAD_VAR, 1, MINUS, RETURN})
}

func TestCompileIf(t *testing.T) {
	tryCompile(t, "(if #t 1 2)", []Word{LOAD_BOOL, Word(0b10011111), POP_JUMP_IF_FALSE, 4, LOAD64, 1 << 2, JUMP_FORWARD, 2, LOAD64, 2 << 2, RETURN})
	tryCompile(t, "(if #t (+ 1 2) 2)", []Word{LOAD_BOOL, Word(0b10011111), POP_JUMP_IF_FALSE, 7, LOAD64, 2 << 2, LOAD64, 1 << 2, PLUS, JUMP_FORWARD, 2, LOAD64, 2 << 2, RETURN})
	tryCompile(t, "(if (= 10 10) (+ 1 2) 2)", []Word{LOAD64, 10 << 2, LOAD64, 10 << 2, EQUALP, POP_JUMP_IF_FALSE, 7, LOAD64, 2 << 2, LOAD64, 1 << 2, PLUS, JUMP_FORWARD, 2, LOAD64, 2 << 2, RETURN})
}

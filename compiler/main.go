package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"
)

type Word uint64

type Parser struct {
	Source string
	Pos    int
	Length int
}

type Compiler struct {
	Code []Word
}

const (
	LOAD64 = iota + 1
	LOAD_BOOL
	LOAD_CHAR
	RETURN
	ADD1
	SUB1
	INTEGER_TO_CHAR
	CHAR_TO_INTEGER
	NULLP
	ZEROP
	NOT
	INTEGERP
	BOOLEANP
	PLUS
	TIMES
	MINUS
	LESS_THAN
	EQUALP
)

const FIXNUM_MASK = 0b11
const FIXNUM_TAG = 0b00
const FIXNUM_SHIFT = 2

const CHAR_MASK = 0b11111111
const CHAR_TAG = 0b00001111
const CHAR_SHIFT = 8

const BOOLEAN_MASK = 0b1111111
const BOOLEAN_TAG = 0b0011111
const BOOLEAN_SHIFT = 7

type Environment interface{}

type Node interface {
	Compile(env Environment) ([]Word, error)
}

type Uint struct {
	Value uint64
}

func (u Uint) Box() (Word, error) {
	if u.Value >= 1<<62 {
		return 0, fmt.Errorf("Compiler does not yet support integers greater than 2^62 - 1")
	}

	return Word(u.Value << FIXNUM_SHIFT), nil
}

func (u Uint) Compile(env Environment) ([]Word, error) {
	boxed, e := u.Box()
	return []Word{LOAD64, boxed}, e
}

type Char struct {
	Value byte
}

func (c Char) Box() (w Word) {
	w = 0

	w += (Word(c.Value) << CHAR_SHIFT)
	w |= CHAR_TAG

	return
}

func (c Char) Compile(env Environment) ([]Word, error) {
	return []Word{LOAD_CHAR, c.Box()}, nil
}

type Boolean struct {
	Value bool
}

func (b Boolean) Box() (w Word) {
	if b.Value {
		w = 1
	} else {
		w = 0
	}

	w <<= BOOLEAN_SHIFT
	w |= BOOLEAN_TAG

	return
}

func (b Boolean) Compile(env Environment) ([]Word, error) {
	return []Word{LOAD_BOOL, b.Box()}, nil
}

type SExp struct {
	Children []Node
}

func (s SExp) Compile(env Environment) ([]Word, error) {
	ws := make([]Word, 0)

	if len(s.Children) == 0 {
		return []Word{}, fmt.Errorf("Found empty SExp")
	}

	switch s.Children[0].(type) {
	case Identifier, SExp:
		break
	default:
		return []Word{}, fmt.Errorf("Invalid functor in S-Expression: %s", s.Children[0]) // TODO: improve message
	}

	for i := len(s.Children) - 1; i >= 0; i-- {
		childrenWords, err := s.Children[i].Compile(env)

		if err != nil {
			return []Word{}, err
		}

		ws = append(ws, childrenWords...)
	}

	return ws, nil
}

type Identifier struct {
	Name string
}

func (i Identifier) Compile(env Environment) ([]Word, error) {
	switch i.Name {
	case "add1":
		return []Word{ADD1}, nil
	case "sub1":
		return []Word{SUB1}, nil
	case "integer->char":
		return []Word{INTEGER_TO_CHAR}, nil
	case "char->integer":
		return []Word{CHAR_TO_INTEGER}, nil
	case "null?":
		return []Word{NULLP}, nil
	case "zero?":
		return []Word{ZEROP}, nil
	case "not":
		return []Word{NOT}, nil
	case "integer?":
		return []Word{INTEGERP}, nil
	case "boolean?":
		return []Word{BOOLEANP}, nil
	case "+":
		return []Word{PLUS}, nil
	case "*":
		return []Word{TIMES}, nil
	case "-":
		return []Word{MINUS}, nil
	case "<":
		return []Word{LESS_THAN}, nil
	case "=":
		return []Word{EQUALP}, nil
	default:
		return []Word{}, fmt.Errorf("Unknown identifier %s", i.Name)
	}
}

func (p Parser) Peek() byte {
	if p.Pos >= p.Length {
		return 0
	}

	return p.Source[p.Pos]
}

func (p *Parser) SkipWhitespace() {
	for unicode.IsSpace(rune(p.Peek())) {
		p.Pos += 1
	}
}

func (p *Parser) ParseNumber() (uint64, error) {
	var number_string strings.Builder

	for unicode.IsDigit(rune(p.Peek())) {
		number_string.WriteRune(rune(p.Peek()))
		p.Pos += 1
	}

	num, e := strconv.ParseUint(number_string.String(), 10, 64)

	return num, e
}

func (p *Parser) ParseIdentifier() (string, error) {
	var nameBuilder strings.Builder

	for !unicode.IsSpace(rune(p.Peek())) && p.Peek() != 0 {
		nameBuilder.WriteByte(p.Peek())
		p.Pos += 1
	}

	return nameBuilder.String(), nil
}

func (p *Parser) ParseSExp() ([]Node, error) {
	if p.Peek() != '(' {
		return []Node{}, fmt.Errorf("Invalid SExp starting with %c", p.Peek())
	}

	p.Pos += 1

	children := []Node{}

	for p.Peek() != ')' && p.Peek() != 0 {
		node, err := p.Parse()

		if err != nil {
			return []Node{}, err
		}

		children = append(children, node)
	}

	if p.Peek() == 0 {
		return []Node{}, fmt.Errorf("Unexpected EOF, expected ')'")
	}

	p.Pos += 1

	return children, nil
}

func (p *Parser) Parse() (Node, error) {
	p.SkipWhitespace()
	c := p.Peek()
	switch c {
	case 0:
		return nil, fmt.Errorf("Unexpected end of input\n")
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		num, e := p.ParseNumber()
		return Uint{num}, e
	case '#':
		p.Pos += 1
		switch p.Peek() {
		case '\\':
			p.Pos += 1

			if p.Peek() != 0 {
				return Char{p.Peek()}, nil
			} else {
				return nil, fmt.Errorf("Unexpected end of input\n")
			}
		case 't':
			return Boolean{true}, nil
		case 'f':
			return Boolean{false}, nil
		default:
			return nil, fmt.Errorf("Unknown literal #%c", p.Peek())
		}
	case '(':
		children, err := p.ParseSExp()

		if err != nil {
			return nil, err
		}

		return SExp{children}, nil
	default:
		name, err := p.ParseIdentifier()

		if err != nil {
			return nil, fmt.Errorf("Unexpected syntax: %s", p.Source[p.Pos:])
		}

		return Identifier{name}, nil
	}
}

func (c *Compiler) Emit(w Word) {
	c.Code = append(c.Code, w)
}

func (c *Compiler) Compile(n Node, e Environment) error {
	code, err := n.Compile(e)

	if err != nil {
		return err
	}

	c.Code = append(c.Code, code...)

	return nil
}

func (c *Compiler) CompileFunction(n Node, e Environment) error {
	err := c.Compile(n, e)

	if err != nil {
		return err
	}

	c.Code = append(c.Code, RETURN)

	return nil
}

func (c Compiler) Serialize(writer io.Writer) {
	for _, w := range c.Code {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(w))
		writer.Write(b)
	}
}

func MakeParser(source string) Parser {
	return Parser{
		source,
		0,
		len(source),
	}
}

func SchemeParse(source string) (Node, error) {
	p := MakeParser(source)
	return p.Parse()
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	source, err := io.ReadAll(os.Stdin)

	check(err)

	program, err := SchemeParse(string(source))

	check(err)

	c := Compiler{}

	err = c.CompileFunction(program, nil)

	check(err)

	c.Serialize(os.Stdout)
}

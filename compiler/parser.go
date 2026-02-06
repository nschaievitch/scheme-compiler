package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Parser struct {
	Source string
	Pos    int
	Length int
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

	for !unicode.IsSpace(rune(p.Peek())) && p.Peek() != ')' && p.Peek() != 0 {
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
		return []Node{}, fmt.Errorf("Unexpected EOF, expected ')', children %v", children)
	}

	p.Pos += 1

	return children, nil
}

func (p *Parser) ParseString() ([]Node, error) {
	if p.Peek() != '"' {
		return []Node{}, fmt.Errorf("Invalid string starting with %c", p.Peek())
	}

	p.Pos += 1

	children := []Node{}

	for p.Peek() != '"' && p.Peek() != 0 {
		if p.Peek() == '\\' {
			p.Pos += 1
			if p.Peek() == 'n' {
				children = append(children, Char{'\n'})
			} else if p.Peek() == 't' {
				children = append(children, Char{'\t'})
			} else {
				children = append(children, Char{p.Peek()})
			}
		} else {
			children = append(children, Char{p.Peek()})
		}

		p.Pos += 1
	}

	if p.Peek() == 0 {
		return []Node{}, fmt.Errorf("Unexpected EOF, expected '\"'")
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
	case '"':
		characters, e := p.ParseString()
		return String{characters}, e
	case '#':
		p.Pos += 1
		switch p.Peek() {
		case '\\':
			p.Pos += 1

			if p.Peek() != 0 {
				c := p.Peek()
				p.Pos += 1
				return Char{c}, nil
			} else {
				p.Pos += 1
				return nil, fmt.Errorf("Unexpected end of input\n")
			}
		case 't':
			p.Pos += 1
			return Boolean{true}, nil
		case 'f':
			p.Pos += 1
			return Boolean{false}, nil
		default:
			return nil, fmt.Errorf("Unknown literal #%c", p.Peek())
		}
	case '(':
		children, err := p.ParseSExp()

		if err != nil {
			return nil, err
		}

		let_id := Identifier{"let"}
		if_id := Identifier{"if"}
		vector_id := Identifier{"vector"}
		string_id := Identifier{"string"}
		begin_id := Identifier{"begin"}

		// test if SExp is let binding
		if len(children) > 0 && children[0] == let_id {
			if len(children) < 3 {
				return nil, fmt.Errorf("Invalid let form, not enough arguments %s", p.Source[p.Pos:])
			}

			bindings := []Binding{}

			for _, b := range children[1 : len(children)-1] {
				switch b.(type) {
				case SExp:
					bs := b.(SExp)

					if len(bs.Children) != 2 {
						return nil, fmt.Errorf("Invalid let binding, must have only two children %v", b)
					}

					switch bs.Children[0].(type) {
					case Identifier:
						bindings = append(bindings, Binding{bs.Children[0].(Identifier), bs.Children[1]})
					default:
						return nil, fmt.Errorf("Invalid let binding, first child should be identifier %v", b)
					}

				default:
					return nil, fmt.Errorf("Invalid let binding %v", b)
				}
			}

			return Let{bindings, children[len(children)-1]}, nil
		}

		if len(children) == 4 && children[0] == if_id {
			return If{
				children[1],
				children[2],
				children[3],
			}, nil
		}

		if len(children) > 0 && children[0] == vector_id {
			return Vector{children[1:]}, nil
		}

		if len(children) > 0 && children[0] == string_id {
			return String{children[1:]}, nil
		}

		if len(children) > 0 && children[0] == begin_id {
			return Begin{children[1:]}, nil
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

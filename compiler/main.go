package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type Word uint64

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
	LOAD_VAR
	POP_JUMP_IF_FALSE
	JUMP_FORWARD
	CONS
	CAR
	CDR
	VECTOR
	VECTOR_REF
	VECTOR_SET
	VECTOR_APPEND
	STRING
	STRING_REF
	STRING_SET
	STRING_APPEND
	SETUP_STACK_BASE
	LABELCALL
	CLOSURE
	FUNCALL
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

type Environment map[Identifier]int

type Node interface {
	Compile(env *Environment, stackIndex *int) ([]Word, error)
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

func (u Uint) Compile(env *Environment, stackIndex *int) ([]Word, error) {
	boxed, e := u.Box()
	*stackIndex += 1
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

func (c Char) Compile(env *Environment, stackIndex *int) ([]Word, error) {
	*stackIndex += 1
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

func (b Boolean) Compile(env *Environment, stackIndex *int) ([]Word, error) {
	*stackIndex += 1
	return []Word{LOAD_BOOL, b.Box()}, nil
}

type SExp struct {
	Children []Node
}

func (s SExp) Compile(env *Environment, stackIndex *int) ([]Word, error) {
	ws := make([]Word, 0)

	if len(s.Children) == 0 {
		return []Word{}, fmt.Errorf("Found empty SExp")
	}

	switch v := s.Children[0].(type) {
	case Identifier:
		if v.Name == "labelcall" {
			ws = append(ws, SETUP_STACK_BASE)

			for i := len(s.Children) - 1; i >= 2; i-- {
				childrenWords, err := s.Children[i].Compile(env, stackIndex)

				if err != nil {
					return []Word{}, err
				}

				ws = append(ws, childrenWords...)
			}

			ws = append(ws, LABELCALL)

			labelIdent, ok := s.Children[1].(Identifier)

			if !ok {
				return []Word{}, fmt.Errorf("Invalid labelcall with non-identifier %v", s.Children[1])
			}

			ws = append(ws, Word((*env)[labelIdent]))

			return ws, nil
		}
	case SExp:
		break
	default:
		return []Word{}, fmt.Errorf("Invalid functor in S-Expression: %s", s.Children[0]) // TODO: improve message
	}

	for i := len(s.Children) - 1; i >= 0; i-- {
		childrenWords, err := s.Children[i].Compile(env, stackIndex)

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

func (i Identifier) Compile(env *Environment, stackIndex *int) ([]Word, error) {
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
		*stackIndex -= 1
		return []Word{PLUS}, nil
	case "*":
		*stackIndex -= 1
		return []Word{TIMES}, nil
	case "-":
		*stackIndex -= 1
		return []Word{MINUS}, nil
	case "<":
		*stackIndex -= 1
		return []Word{LESS_THAN}, nil
	case "=":
		*stackIndex -= 1
		return []Word{EQUALP}, nil
	case "cons":
		*stackIndex -= 1
		return []Word{CONS}, nil
	case "car":
		return []Word{CAR}, nil
	case "cdr":
		return []Word{CDR}, nil
	case "vector-ref":
		*stackIndex -= 1
		return []Word{VECTOR_REF}, nil
	case "vector-set!":
		*stackIndex -= 1 // returns a new pointer to the vector?
		return []Word{VECTOR_SET}, nil
	case "vector-append":
		*stackIndex -= 1
		return []Word{VECTOR_APPEND}, nil
	case "string-ref":
		*stackIndex -= 1
		return []Word{STRING_REF}, nil
	case "string-set!":
		*stackIndex -= 1
		return []Word{STRING_SET}, nil
	case "string-append":
		*stackIndex -= 1
		return []Word{STRING_APPEND}, nil
	case "labelcall":
		return []Word{LABELCALL}, nil
	default:
		si, ok := (*env)[i]

		if !ok {
			return []Word{}, fmt.Errorf("Unknown identifier %s", i.Name)
		}

		*stackIndex += 1
		return []Word{LOAD_VAR, Word(si)}, nil
	}
}

type Binding struct {
	Identifier Identifier
	Value      Node
}

type Let struct {
	Bindings []Binding
	Body     Node
}

func (l Let) Compile(env *Environment, stackIndex *int) ([]Word, error) {
	code := []Word{}
	// Evaluate bindings and save stack positions
	for _, b := range l.Bindings {
		val, err := b.Value.Compile(env, stackIndex)

		if err != nil {
			return []Word{}, err
		}

		code = append(code, val...)

		(*env)[b.Identifier] = *stackIndex - 1
	}

	body, err := l.Body.Compile(env, stackIndex)

	if err != nil {
		return []Word{}, err
	}

	code = append(code, body...)

	return code, nil
}

type If struct {
	Condition   Node
	Consequent  Node
	Alternative Node
}

func (i If) Compile(env *Environment, stackIndex *int) ([]Word, error) {
	cond, err := i.Condition.Compile(env, stackIndex)

	if err != nil {
		return []Word{}, err
	}

	cons, err := i.Consequent.Compile(env, stackIndex)

	if err != nil {
		return []Word{}, err
	}

	alt, err := i.Alternative.Compile(env, stackIndex)

	if err != nil {
		return []Word{}, err
	}

	code := []Word{}

	// Run condition
	code = append(code, cond...)

	// Evaluate and jump accordingly
	// We use offset from current PC
	code = append(code, POP_JUMP_IF_FALSE)

	code = append(code, Word(len(cons)+2))

	code = append(code, cons...)

	code = append(code, JUMP_FORWARD)

	code = append(code, Word(len(alt)))

	code = append(code, alt...)

	return code, nil
}

type Vector struct {
	Children []Node
}

func (v Vector) Compile(env *Environment, stackIndex *int) ([]Word, error) {
	code := []Word{}

	for i := len(v.Children) - 1; i >= 0; i-- {
		words, err := v.Children[i].Compile(env, stackIndex)

		if err != nil {
			return []Word{}, err
		}

		code = append(code, words...)
	}

	code = append(code, VECTOR, Word(len(v.Children)))

	*stackIndex -= len(v.Children) - 1

	return code, nil
}

type String struct {
	Children []Node
}

func (s String) Compile(env *Environment, stackIndex *int) ([]Word, error) {
	code := []Word{}

	for i := len(s.Children) - 1; i >= 0; i-- {
		if _, ok := s.Children[i].(Char); !ok {
			return []Word{}, fmt.Errorf("Non character in string call: %v", s.Children)
		}

		words, err := s.Children[i].Compile(env, stackIndex)

		if err != nil {
			return []Word{}, err
		}

		code = append(code, words...)
	}

	code = append(code, STRING, Word(len(s.Children)))

	*stackIndex -= len(s.Children) - 1

	return code, nil
}

type Begin struct {
	Children []Node
}

func (b Begin) Compile(env *Environment, stackIndex *int) ([]Word, error) {
	code := []Word{}

	for _, c := range b.Children {
		words, err := c.Compile(env, stackIndex)

		if err != nil {
			return []Word{}, err
		}

		code = append(code, words...)
	}

	return code, nil
}

type Code struct {
	Vars       []Identifier
	FreeVars   []Identifier
	Expression Node
}

func (c Code) Compile(env *Environment, stackIndex *int) ([]Word, error) {
	// make local environment, assuming the variables are pushed into the stack in reverse order
	localEnv := make(Environment, len(*env))

	for k, v := range *env {
		localEnv[k] = v
	}

	for i, v := range c.Vars {
		localEnv[v] = len(c.Vars) - i - 1 // vars should be in %ebp + varNum, in reverse order
	}

	localStackIndex := 0

	code, err := c.Expression.Compile(&localEnv, &localStackIndex)

	code = append(code, RETURN)

	if err != nil {
		return []Word{}, err
	}

	return code, nil
}

type Label struct {
	Identifier Identifier
	Code       Code
}

type Labels struct {
	LabelList  []Label
	Expression Node
}

// assumes it is top level expr
// to compile each label with the lvars already in the environment in one pass, we store the codes like so:
// the bytecode starts with a subarray of code positions:
// [main pos ; f1 pos ; ...]
// followed by the actual codes, whose length we don't know until compiling
// When doing a label call, we jump to code[label] instead of label
// This adds one level of indirection, but performance hit should be negligible
func (l Labels) Compile(env *Environment, stackIndex *int) ([]Word, error) {
	code := make([]Word, len(l.LabelList)+1) // we store the "main" first

	for i, label := range l.LabelList {
		(*env)[label.Identifier] = i + 1
	}

	for i, label := range l.LabelList {
		code[i+1] = Word(len(code))

		words, err := label.Code.Compile(env, stackIndex)

		if err != nil {
			return []Word{}, err
		}

		code = append(code, words...)
	}

	code[0] = Word(len(code))

	words, err := l.Expression.Compile(env, stackIndex)

	if err != nil {
		return []Word{}, err
	}

	code = append(code, words...)

	return code, nil
}

type Closure struct {
	LVar   Identifier
	Values []Node
}

func (c Closure) Compile(env *Environment, stackIndex *int) ([]Word, error) {
	label, ok := (*env)[c.LVar]

	if !ok {
		return []Word{}, fmt.Errorf("Unknown label %s in closure", c.LVar)
	}

	code := make([]Word, 0)

	for i := len(c.Values) - 1; i >= 0; i-- {
		words, err := c.Values[i].Compile(env, stackIndex)

		if err != nil {
			return []Word{}, err
		}

		code = append(code, words...)
	}

	code = append(code, CLOSURE, Word(label), Word(len(c.Values)))

	*stackIndex -= len(c.Values) - 1

	return code, nil

}

func (c *Compiler) Emit(w Word) {
	c.Code = append(c.Code, w)
}

func (c *Compiler) Compile(n Node, e *Environment) error {
	var stackIndex int

	// We keep positive stack indexes instead of negative (counting from the bottom of the stack)
	// We also count in 64bit words, not bytes
	stackIndex = 0

	// if no toplevel labels, push 1 first for main
	if _, ok := n.(Labels); !ok {
		c.Code = append(c.Code, 1)
	}

	code, err := n.Compile(e, &stackIndex)

	if err != nil {
		return err
	}

	c.Code = append(c.Code, code...)

	return nil
}

func (c *Compiler) CompileFunction(n Node, e *Environment) error {
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

	env := make(Environment)

	err = c.CompileFunction(program, &env)

	check(err)

	c.Serialize(os.Stdout)
}

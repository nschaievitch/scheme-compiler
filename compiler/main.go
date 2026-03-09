package main

import (
	_ "embed"
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
	LOAD_CLOSURE_VAR
	SLIDE
	TAILCALL
	LOAD_VAR_DEFER
	SET_DEFERRED
	PUSH_NIL
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

//go:embed stdlib.scm
var STDLIB_SCM string

type VariableResolution struct {
	Offset         int
	IsClosureVar   bool
	IsDeferredLoad bool
}
type Environment map[Identifier]VariableResolution

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

			ws = append(ws, Word((*env)[labelIdent].Offset))

			*stackIndex -= len(s.Children) - 2
			return ws, nil
		} else if v.Name == "funcall" || v.Name == "tailcall" { // ugly repeated code... should improve...
			ws = append(ws, SETUP_STACK_BASE)

			for i := len(s.Children) - 1; i >= 1; i-- {
				childrenWords, err := s.Children[i].Compile(env, stackIndex)

				if err != nil {
					return []Word{}, err
				}

				ws = append(ws, childrenWords...)
			}

			if v.Name == "funcall" {
				ws = append(ws, FUNCALL)
			} else {
				ws = append(ws, TAILCALL)
				ws = append(ws, Word((len(s.Children) - 2)))
			}

			*stackIndex -= len(s.Children) - 2
			return ws, nil

		} else if _, ok := BUILTINS[v.Name]; !ok { // if it is a variable, do implicit funcall
			ws = append(ws, SETUP_STACK_BASE)

			for i := len(s.Children) - 1; i >= 0; i-- {
				childrenWords, err := s.Children[i].Compile(env, stackIndex)

				if err != nil {
					return []Word{}, err
				}

				ws = append(ws, childrenWords...)
			}

			ws = append(ws, FUNCALL)

			*stackIndex -= len(s.Children) - 1
			return ws, nil

		}
	case SExp, Closure:
		// if the first element is a SExp or closure, we do an implicit funcall
		ws = append(ws, SETUP_STACK_BASE)

		for i := len(s.Children) - 1; i >= 0; i-- {
			childrenWords, err := s.Children[i].Compile(env, stackIndex)

			if err != nil {
				return []Word{}, err
			}

			ws = append(ws, childrenWords...)
		}

		ws = append(ws, FUNCALL)

		*stackIndex -= len(s.Children) - 1
		return ws, nil
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
	case "'()":
		*stackIndex += 1
		return []Word{PUSH_NIL}, nil
	default:
		varRes, ok := (*env)[i]

		if !ok {
			return []Word{}, fmt.Errorf("Unknown identifier %s", i.Name)
		}

		*stackIndex += 1

		if varRes.IsClosureVar {
			return []Word{LOAD_CLOSURE_VAR, Word(varRes.Offset)}, nil

		}

		if varRes.IsDeferredLoad {
			return []Word{LOAD_VAR_DEFER, Word(varRes.Offset)}, nil
		}

		return []Word{LOAD_VAR, Word(varRes.Offset)}, nil
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
	localEnv := make(Environment)
	for _, b := range l.Bindings {
		val, err := b.Value.Compile(env, stackIndex)

		if err != nil {
			return []Word{}, err
		}

		code = append(code, val...)

		localEnv[b.Identifier] = VariableResolution{*stackIndex - 1, false, false}
	}

	for k, v := range *env {
		_, ok := localEnv[k]

		if !ok {
			localEnv[k] = v
		}
	}

	body, err := l.Body.Compile(&localEnv, stackIndex)

	if err != nil {
		return []Word{}, err
	}

	code = append(code, body...)
	code = append(code, SLIDE, Word(len(l.Bindings)))
	*stackIndex -= len(l.Bindings)

	return code, nil
}

type LetRec struct {
	Bindings []Binding
	Body     Node
}

func (l LetRec) Compile(env *Environment, stackIndex *int) ([]Word, error) {
	code := []Word{}
	// Evaluate bindings and save stack positions
	for _, b := range l.Bindings {
		_, ok := b.Value.(Closure)

		if !ok {
			return nil, fmt.Errorf("Not implemented: letrec can only be used with lambdas")
		}

		(*env)[b.Identifier] = VariableResolution{*stackIndex, false, true}

		*stackIndex += 1
	}

	deferredCount := 0
	for _, b := range l.Bindings {
		if closure, ok := b.Value.(Closure); ok {
			for _, v := range closure.Values {
				if id, ok := v.(Identifier); ok {
					if res, ok := (*env)[id]; ok && res.IsDeferredLoad {
						deferredCount++
					}
				}
			}
		}
	}

	for _, b := range l.Bindings {
		val, err := b.Value.Compile(env, stackIndex)

		if err != nil {
			return []Word{}, err
		}

		code = append(code, val...)
	}

	code = append(code, SET_DEFERRED, Word(deferredCount))

	for _, b := range l.Bindings {
		v := (*env)[b.Identifier]
		v.IsDeferredLoad = false
		(*env)[b.Identifier] = v
	}

	body, err := l.Body.Compile(env, stackIndex)

	if err != nil {
		return []Word{}, err
	}

	code = append(code, body...)
	code = append(code, SLIDE, Word(len(l.Bindings)))
	*stackIndex -= len(l.Bindings)

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

func CopyMap[K comparable, V any](src map[K]V) map[K]V {
	if src == nil {
		return nil
	}
	dst := make(map[K]V, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func (c Code) Compile(env *Environment, stackIndex *int) ([]Word, error) {
	// make local environment, assuming the variables are pushed into the stack in reverse order
	var localEnv Environment
	localEnv = CopyMap(*env)

	for i, v := range c.Vars {
		localEnv[v] = VariableResolution{len(c.Vars) - i - 1, false, false} // vars should be in %ebp + varNum, in reverse order
	}

	for i, v := range c.FreeVars {
		localEnv[v] = VariableResolution{i, true, false} // free vars are taken from the closure
	}

	localStackIndex := len(c.Vars)

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
		(*env)[label.Identifier] = VariableResolution{i + 1, false, false}
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

	code = append(code, CLOSURE, Word(label.Offset), Word(len(c.Values)))

	*stackIndex -= len(c.Values) - 1

	return code, nil

}

type Lambda struct {
	Args []Identifier
	Code Node
}

func (l Lambda) Compile(env *Environment, stackIndex *int) ([]Word, error) {
	panic("Attempted to compile lambda without lifting")
}

// i found eta expansion to be the correct name for this, but not 100% sure
func etaExpandIdent(name string) Node {
	arity := BUILTINS[name]
	args := make([]Identifier, arity)
	argNodes := make([]Node, arity+1)
	argNodes[0] = Identifier{name}
	for i := 0; i < arity; i++ {
		id := Identifier{fmt.Sprintf("__eta%d", i)}
		args[i] = id
		argNodes[i+1] = id
	}
	return Lambda{Args: args, Code: SExp{Children: argNodes}}
}

func EtaExpandBuiltins(n Node) Node {
	switch v := n.(type) {
	case SExp:
		children := make([]Node, len(v.Children))
		children[0] = v.Children[0]

		for i := 1; i < len(v.Children); i++ {
			child := EtaExpandBuiltins(v.Children[i])

			if id, ok := child.(Identifier); ok {
				if _, isBlt := BUILTINS[id.Name]; isBlt && id.Name != "funcall" && id.Name != "tailcall" && id.Name != "labelcall" && id.Name != "'()" {
					child = etaExpandIdent(id.Name)
				}
			}

			children[i] = child
		}
		return SExp{Children: children}
	case Let:
		binds := make([]Binding, len(v.Bindings))

		for i, b := range v.Bindings {
			binds[i] = Binding{b.Identifier, EtaExpandBuiltins(b.Value)}
		}

		return Let{Bindings: binds, Body: EtaExpandBuiltins(v.Body)}
	case LetRec:
		binds := make([]Binding, len(v.Bindings))

		for i, b := range v.Bindings {
			binds[i] = Binding{b.Identifier, EtaExpandBuiltins(b.Value)}
		}

		return LetRec{Bindings: binds, Body: EtaExpandBuiltins(v.Body)}
	case If:
		return If{
			Condition:   EtaExpandBuiltins(v.Condition),
			Consequent:  EtaExpandBuiltins(v.Consequent),
			Alternative: EtaExpandBuiltins(v.Alternative),
		}
	case Lambda:
		return Lambda{Args: v.Args, Code: EtaExpandBuiltins(v.Code)}
	case Begin:
		children := make([]Node, len(v.Children))

		for i, e := range v.Children {
			children[i] = EtaExpandBuiltins(e)
		}
		return Begin{Children: children}
	case Vector:
		children := make([]Node, len(v.Children))

		for i, e := range v.Children {
			children[i] = EtaExpandBuiltins(e)
		}
		return Vector{Children: children}
	default:
		return n
	}
}

var BUILTINS = map[string]int{
	"add1":          1,
	"sub1":          1,
	"integer->char": 1,
	"char->integer": 1,
	"null?":         1,
	"zero?":         1,
	"not":           1,
	"integer?":      1,
	"boolean?":      1,
	"+":             2,
	"*":             2,
	"-":             2,
	"<":             2,
	"=":             2,
	"cons":          2,
	"car":           1,
	"cdr":           1,
	"vector-ref":    2,
	"vector-set!":   3,
	"vector-append": 2,
	"string-ref":    2,
	"string-set!":   3,
	"string-append": 2,
	"funcall":       1,
	"tailcall":      1,
	"labelcall":     1,
	"'()":           0,
}

func LiftLambdasRec(n Node, bound map[Identifier]bool, labels *[]Label) (transformed Node, free map[Identifier]bool) {
	boundLocal := CopyMap(bound)
	free = make(map[Identifier]bool)

	switch v := n.(type) {
	case Let:
		binds := make([]Binding, 0)

		for _, b := range v.Bindings {
			tr, fr := LiftLambdasRec(b.Value, boundLocal, labels)

			for k := range fr {
				if _, ok := boundLocal[k]; !ok {
					free[k] = true
				}
			}

			binds = append(binds, Binding{b.Identifier, tr})
			boundLocal[b.Identifier] = true
		}

		tr, fr := LiftLambdasRec(v.Body, boundLocal, labels)

		for k := range fr {
			if _, ok := boundLocal[k]; !ok {
				free[k] = true
			}
		}

		return Let{
			Bindings: binds,
			Body:     tr,
		}, free
	case LetRec:
		binds := make([]Binding, 0)

		for _, b := range v.Bindings {
			boundLocal[b.Identifier] = true
		}
		for _, b := range v.Bindings {
			tr, fr := LiftLambdasRec(b.Value, boundLocal, labels)

			for k := range fr {
				if _, ok := boundLocal[k]; !ok {
					free[k] = true
				}
			}

			binds = append(binds, Binding{b.Identifier, tr})
		}

		tr, fr := LiftLambdasRec(v.Body, boundLocal, labels)

		for k := range fr {
			if _, ok := boundLocal[k]; !ok {
				free[k] = true
			}
		}

		return LetRec{
			Bindings: binds,
			Body:     tr,
		}, free
	case Lambda:
		boundLocal = make(map[Identifier]bool)
		for _, arg := range v.Args {
			boundLocal[arg] = true
		}

		tr, fr := LiftLambdasRec(v.Code, boundLocal, labels)

		freeVars := make([]Identifier, 0)

		for k := range fr {
			if _, ok := boundLocal[k]; !ok {
				free[k] = true
				freeVars = append(freeVars, k)
			}
		}

		labName := Identifier{fmt.Sprintf("__f%d", len(*labels))}
		lab := Label{
			Identifier: labName,
			Code: Code{
				Vars:       v.Args,
				FreeVars:   freeVars,
				Expression: tr,
			},
		}

		*labels = append(*labels, lab)

		// sadly can't simply upcast apparently
		freeVarNodes := make([]Node, 0)

		for _, vr := range freeVars {
			freeVarNodes = append(freeVarNodes, vr)
		}

		return Closure{
			LVar:   labName,
			Values: freeVarNodes,
		}, free

	case Identifier:
		_, ok := bound[v]

		if !ok {
			_, ok = BUILTINS[v.Name]

			if !ok {
				free[v] = true
			}
		}

		return n, free
	case SExp:
		children := make([]Node, 0)

		for _, c := range v.Children {
			tr, fr := LiftLambdasRec(c, boundLocal, labels)

			for k := range fr {
				free[k] = true
			}

			children = append(children, tr)
		}

		return SExp{children}, free
	case Vector:
		children := make([]Node, 0)

		for _, c := range v.Children {
			tr, fr := LiftLambdasRec(c, boundLocal, labels)

			for k := range fr {
				free[k] = true
			}

			children = append(children, tr)
		}

		return Vector{children}, free
	case Closure:
		values := make([]Node, 0)
		for _, c := range v.Values {
			tr, fr := LiftLambdasRec(c, boundLocal, labels)

			for k := range fr {
				free[k] = true
			}

			values = append(values, tr)
		}
		return Closure{v.LVar, values}, free
	case Begin:
		children := make([]Node, 0)
		for _, c := range v.Children {
			tr, fr := LiftLambdasRec(c, boundLocal, labels)

			for k := range fr {
				free[k] = true
			}

			children = append(children, tr)
		}
		return Begin{children}, free
	case If:
		tr_condition, fr := LiftLambdasRec(v.Condition, bound, labels)

		for k := range fr {
			free[k] = true
		}

		tr_consequent, fr := LiftLambdasRec(v.Consequent, bound, labels)

		for k := range fr {
			free[k] = true
		}

		tr_alternative, fr := LiftLambdasRec(v.Alternative, bound, labels)

		for k := range fr {
			free[k] = true
		}

		return If{Condition: tr_condition, Consequent: tr_consequent, Alternative: tr_alternative}, free
	default:
		return n, free
	}
}

func LiftLambdas(program Node) (Node, error) {
	// The compiler does not support labels being used with lambdas, UB if both are used
	if _, ok := program.(Labels); ok {
		return program, nil
	}

	bound := make(map[Identifier]bool)
	labels := make([]Label, 0)

	tr, free := LiftLambdasRec(program, bound, &labels)

	if len(free) > 0 {
		return nil, fmt.Errorf("Found free variable(s) in top level program: %v", free)
	}

	return Labels{LabelList: labels, Expression: tr}, nil
}

func TailCallOptimize(n Node, isTail bool) (Node, error) {
	switch v := n.(type) {
	case Labels:
		exp, err := TailCallOptimize(v.Expression, true)

		if err != nil {
			return nil, err
		}

		labels := make([]Label, 0)

		for _, l := range v.LabelList {
			nl, err := TailCallOptimize(l.Code, true)

			if err != nil {
				return nil, err
			}

			labels = append(labels, Label{l.Identifier, nl.(Code)})
		}

		return Labels{LabelList: labels, Expression: exp}, nil
	case SExp:
		_, isClosure := v.Children[0].(Closure)
		id, isIdent := v.Children[0].(Identifier)

		_, builtin := BUILTINS[id.Name]

		if builtin {
			isIdent = false
		}

		ch := make([]Node, 0)

		for _, c := range v.Children {
			tc, err := TailCallOptimize(c, false)

			if err != nil {
				return nil, err
			}

			ch = append(ch, tc)
		}

		if (isClosure || isIdent) && isTail {
			ch = append([]Node{Identifier{"tailcall"}}, ch...)
		}

		return SExp{ch}, nil

	case Let:
		body, err := TailCallOptimize(v.Body, isTail)

		if err != nil {
			return nil, err
		}

		bindings := make([]Binding, 0)

		for _, b := range v.Bindings {
			nv, err := TailCallOptimize(b.Value, false) // could it be true?

			if err != nil {
				return nil, err
			}

			bindings = append(bindings, Binding{b.Identifier, nv})
		}

		return Let{
			Bindings: bindings,
			Body:     body,
		}, nil
	case LetRec:
		body, err := TailCallOptimize(v.Body, isTail)

		if err != nil {
			return nil, err
		}

		bindings := make([]Binding, 0)

		for _, b := range v.Bindings {
			nv, err := TailCallOptimize(b.Value, false) // could it be true?

			if err != nil {
				return nil, err
			}

			bindings = append(bindings, Binding{b.Identifier, nv})
		}

		return LetRec{
			Bindings: bindings,
			Body:     body,
		}, nil
	case If:
		cond, err := TailCallOptimize(v.Condition, false)

		if err != nil {
			return nil, err
		}

		cons, err := TailCallOptimize(v.Consequent, isTail)

		if err != nil {
			return nil, err
		}

		alt, err := TailCallOptimize(v.Alternative, isTail)

		if err != nil {
			return nil, err
		}

		return If{
			Condition:   cond,
			Consequent:  cons,
			Alternative: alt,
		}, nil
	case Begin:
		ch := make([]Node, 0)

		for i, c := range v.Children {
			nc, err := TailCallOptimize(c, (i == len(v.Children)-1 && isTail))

			if err != nil {
				return nil, err
			}

			ch = append(ch, nc)
		}

		return Begin{ch}, nil
	case Code:
		exp, err := TailCallOptimize(v.Expression, isTail)

		if err != nil {
			return nil, err
		}

		return Code{
			Vars:       v.Vars,
			FreeVars:   v.FreeVars,
			Expression: exp,
		}, nil
	}

	return n, nil
}

func AddStdlib(program Node) (Node, error) {
	// for testing only, we don't add stdlib if topform is labels
	if _, ok := program.(Labels); ok {
		return program, nil
	}

	stdlib, err := SchemeParse(STDLIB_SCM)

	if err != nil {
		return nil, fmt.Errorf("Error parsing stdlib: %e\n", err)
	}

	stdlib_letrec, ok := stdlib.(LetRec)

	if !ok {
		return nil, fmt.Errorf("Stdlib is not letrec\n")
	}

	return LetRec{append([]Binding{{Identifier{
		"main",
	}, Lambda{[]Identifier{}, program}}}, stdlib_letrec.Bindings...), stdlib_letrec.Body}, nil
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

	linkedProgram, err := AddStdlib(program)

	check(err)

	expandedProgram := EtaExpandBuiltins(linkedProgram)

	liftedProgram, err := LiftLambdas(expandedProgram)

	check(err)

	tailCalled, err := TailCallOptimize(liftedProgram, true)

	check(err)

	// panic(fmt.Errorf("%+v %+v\n", liftedProgram, tailCalled))

	c := Compiler{}

	env := make(Environment)

	err = c.CompileFunction(tailCalled, &env)

	check(err)

	c.Serialize(os.Stdout)
}

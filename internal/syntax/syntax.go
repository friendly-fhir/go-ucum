package syntax

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
)

var (
	// ErrInvalidSyntax reports an expression that does not follow the UCUM
	// grammar.
	ErrInvalidSyntax = errors.New("ucum: invalid syntax")

	// ErrUnknownUnit reports a symbol that names no unit atom, or a prefix
	// applied to an atom that does not accept prefixes.
	ErrUnknownUnit = errors.New("ucum: unknown unit")
)

// Lookup resolves symbols in one set of definitions and one case mode.
type Lookup interface {
	// Atom returns the identity of the unit atom with the given code and
	// whether that atom accepts prefixes.
	Atom(code string) (id int, metric bool, ok bool)

	// Prefix returns the identity of the prefix with the given code.
	Prefix(code string) (id int, ok bool)
}

// Node is an element of a parsed unit expression.
type Node interface {
	node()
}

// Op is a binary operator in a unit term.
type Op byte

// The binary operators of a unit term. Both have the same precedence and
// associate to the left.
const (
	Multiply Op = '.'
	Divide   Op = '/'
)

// Binary is a product or quotient of two terms.
type Binary struct {
	Op    Op
	Left  Node
	Right Node
}

// Inverse is a term preceded by the unary division operator, as in "/s".
type Inverse struct {
	Operand Node
}

// Group is a parenthesized term.
type Group struct {
	Term Node
}

// Factor is a positive integer written in place of a unit, as in "/24".
type Factor struct {
	Value *big.Int
}

// Unity is an annotation that stands alone, as in "{cells}", and denotes the
// number one.
type Unity struct{}

// Symbol is a unit atom with an optional prefix, exponent, and annotation.
type Symbol struct {
	// Offset is the byte offset of the symbol in the input.
	Offset int

	HasPrefix bool
	Prefix    int
	Atom      int

	HasExponent bool
	Exponent    int
}

func (Binary) node()  {}
func (Inverse) node() {}
func (Group) node()   {}
func (Factor) node()  {}
func (Unity) node()   {}
func (Symbol) node()  {}

var (
	_ Node = Binary{}
	_ Node = Inverse{}
	_ Node = Group{}
	_ Node = Factor{}
	_ Node = Unity{}
	_ Node = Symbol{}
)

// Error describes where and why an expression could not be parsed.
//
// Err wraps [ErrInvalidSyntax] or [ErrUnknownUnit].
type Error struct {
	Offset int
	Err    error
}

func (e *Error) Error() string {
	return fmt.Sprintf("offset %d: %v", e.Offset, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

var _ error = (*Error)(nil)

// Parse parses input as a UCUM main term, resolving symbols with lookup.
//
// Parse returns an [*Error] wrapping [ErrInvalidSyntax] if input does not
// follow the grammar or contains an exponent whose magnitude exceeds
// maxExponent, and wrapping [ErrUnknownUnit] if a symbol cannot be
// resolved. Bounding exponents bounds the size of the numbers that
// interpreting the result can produce.
func Parse(input string, lookup Lookup, maxExponent int) (Node, error) {
	p := &parser{input: input, lookup: lookup, maxExponent: maxExponent}
	if input == "" {
		return nil, p.errorf(0, ErrInvalidSyntax, "empty expression")
	}

	var n Node
	var err error
	if p.peek() == '/' {
		p.pos++
		n, err = p.term()
		n = Inverse{Operand: n}
	} else {
		n, err = p.term()
	}
	if err != nil {
		return nil, err
	}
	if p.pos < len(input) {
		return nil, p.errorf(p.pos, ErrInvalidSyntax, "unexpected %q", input[p.pos])
	}
	return n, nil
}

type parser struct {
	input       string
	pos         int
	lookup      Lookup
	maxExponent int
}

func (p *parser) peek() byte {
	if p.pos < len(p.input) {
		return p.input[p.pos]
	}
	return 0
}

func (p *parser) errorf(offset int, sentinel error, format string, args ...any) error {
	return &Error{Offset: offset, Err: fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)}
}

func (p *parser) term() (Node, error) {
	left, err := p.component()
	if err != nil {
		return nil, err
	}
	for {
		op := Op(p.peek())
		if op != Multiply && op != Divide {
			return left, nil
		}
		p.pos++
		right, err := p.component()
		if err != nil {
			return nil, err
		}
		left = Binary{Op: op, Left: left, Right: right}
	}
}

func (p *parser) component() (Node, error) {
	switch p.peek() {
	case '(':
		open := p.pos
		p.pos++
		t, err := p.term()
		if err != nil {
			return nil, err
		}
		if p.peek() != ')' {
			return nil, p.errorf(open, ErrInvalidSyntax, "unclosed parenthesis")
		}
		p.pos++
		return Group{Term: t}, nil
	case '{':
		if err := p.annotation(); err != nil {
			return nil, err
		}
		return Unity{}, nil
	}

	start := p.pos
	token, err := p.symbolToken()
	if err != nil {
		return nil, err
	}
	if token == "" {
		if p.pos == len(p.input) {
			return nil, p.errorf(p.pos, ErrInvalidSyntax, "expected a unit at end of expression")
		}
		return nil, p.errorf(p.pos, ErrInvalidSyntax, "expected a unit before %q", p.input[p.pos])
	}
	if p.isDigits(token) {
		v, _ := new(big.Int).SetString(token, 10)
		if v.Sign() == 0 {
			return nil, p.errorf(start, ErrInvalidSyntax, "factor %q is not a positive integer", token)
		}
		return Factor{Value: v}, nil
	}

	sym, err := p.symbol(start, token)
	if err != nil {
		return nil, err
	}
	if p.peek() == '{' {
		if err := p.annotation(); err != nil {
			return nil, err
		}
	}
	return sym, nil
}

// symbolToken consumes the text of a prefixed atom with its exponent: every
// character up to the next operator, parenthesis, or annotation. Square
// brackets are matched, and their contents are taken literally.
func (p *parser) symbolToken() (string, error) {
	start := p.pos
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		switch {
		case c == '.' || c == '/' || c == '(' || c == ')' || c == '{' || c == '}':
			return p.input[start:p.pos], nil
		case c == '[':
			if err := p.bracket(); err != nil {
				return "", err
			}
			continue
		case c == ']':
			return "", p.errorf(p.pos, ErrInvalidSyntax, "unmatched ']'")
		case !p.isPrintable(c):
			return "", p.errorf(p.pos, ErrInvalidSyntax, "invalid character %q", c)
		}
		p.pos++
	}
	return p.input[start:], nil
}

func (p *parser) bracket() error {
	open := p.pos
	for p.pos++; p.pos < len(p.input); p.pos++ {
		switch c := p.input[p.pos]; {
		case c == ']':
			p.pos++
			return nil
		case c == '[':
			return p.errorf(p.pos, ErrInvalidSyntax, "nested '['")
		case !p.isPrintable(c):
			return p.errorf(p.pos, ErrInvalidSyntax, "invalid character %q", c)
		}
	}
	return p.errorf(open, ErrInvalidSyntax, "unclosed '['")
}

func (p *parser) annotation() error {
	open := p.pos
	for p.pos++; p.pos < len(p.input); p.pos++ {
		switch c := p.input[p.pos]; {
		case c == '}':
			p.pos++
			return nil
		case c == '{':
			return p.errorf(p.pos, ErrInvalidSyntax, "nested '{'")
		case !p.isPrintable(c):
			return p.errorf(p.pos, ErrInvalidSyntax, "invalid character %q", c)
		}
	}
	return p.errorf(open, ErrInvalidSyntax, "unclosed '{'")
}

// symbol splits token into a prefixed atom and an exponent and resolves the
// atom and prefix. An atom never ends in a digit, so a trailing run of digits,
// optionally signed, is always the exponent.
func (p *parser) symbol(offset int, token string) (Symbol, error) {
	sym := Symbol{Offset: offset}

	end := len(token)
	for end > 0 && p.isDigit(token[end-1]) {
		end--
	}
	if end < len(token) {
		digits := end
		if end > 0 && (token[end-1] == '+' || token[end-1] == '-') {
			end--
		}
		if end == 0 {
			return sym, p.errorf(offset, ErrInvalidSyntax, "exponent %q without a unit", token)
		}
		exp, err := strconv.Atoi(token[end:])
		if err != nil || exp > p.maxExponent || exp < -p.maxExponent {
			return sym, p.errorf(offset+digits, ErrInvalidSyntax, "exponent %s exceeds the limit of %d", token[end:], p.maxExponent)
		}
		sym.HasExponent, sym.Exponent = true, exp
	}

	code := token[:end]
	if id, _, ok := p.lookup.Atom(code); ok {
		sym.Atom = id
		return sym, nil
	}
	// The longest prefix that leaves a metric atom wins.
	for split := len(code) - 1; split > 0; split-- {
		prefix, ok := p.lookup.Prefix(code[:split])
		if !ok {
			continue
		}
		if atom, metric, ok := p.lookup.Atom(code[split:]); ok && metric {
			sym.HasPrefix, sym.Prefix, sym.Atom = true, prefix, atom
			return sym, nil
		}
	}
	return sym, p.errorf(offset, ErrUnknownUnit, "%q", code)
}

func (*parser) isDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

func (p *parser) isDigits(s string) bool {
	for i := range len(s) {
		if !p.isDigit(s[i]) {
			return false
		}
	}
	return true
}

// isPrintable reports whether c may appear in a UCUM expression: the printable
// ASCII characters other than space.
func (*parser) isPrintable(c byte) bool {
	return '!' <= c && c <= '~'
}

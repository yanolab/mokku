package mokku

import (
	"errors"
	"fmt"
	"go/scanner"
	"go/token"
	"strings"
)

type targetInterface struct {
	name    string
	methods []*method
}

type method struct {
	name string

	// e.g. '( foo , bar string ) ( a int , err error )'
	// extracting it all as a string to keep things simple while still being
	// able flexible.
	signature string
}

func (m *method) String() string {
	return fmt.Sprintf("<name: %s, signature: %s>", m.name, m.signature)
}

type parser struct {
	s   *scanner.Scanner
	src []byte
}

func newParser(src []byte) *parser {
	var (
		fs = token.NewFileSet()
		s  = &scanner.Scanner{}
	)
	// currently the only scanner.Mode option is to include
	// comments (and a private option for not including semicolons).
	// Selecting 0 to *not* scan comments.
	bareEssentials := scanner.Mode(0)
	s.Init(fs.AddFile("", fs.Base(), len(src)), src, nil, bareEssentials)
	return &parser{s: s, src: src}
}

func (p *parser) parse() (*targetInterface, error) {
	name, err := p.lookForItfcName()
	if err != nil {
		return nil, fmt.Errorf("unable to find interface name in '%s':%w", p.src, err)
	}

	methods, err := p.lookForMethods()
	if err != nil {
		return nil, err
	}

	return &targetInterface{name: name, methods: methods}, nil
}

func (p *parser) lookForItfcName() (string, error) {
	for {
		_, tok, _ := p.s.Scan()
		switch tok {
		case token.EOF:
			return "", errors.New("unable to find interface declaration")
		case token.INTERFACE:
			return "", nil // Treat as an anonymous interface
		case token.TYPE: // <type> Foo interface
			_, tok, lit := p.s.Scan()
			if tok == token.IDENT { // type <Foo> interface
				_, tok, _ := p.s.Scan()
				if tok == token.INTERFACE { // type Foo <interface>
					return lit, nil
				}
			}
		}
	}
}

func (p *parser) lookForMethods() ([]*method, error) {
	for {
		_, tok, _ := p.s.Scan()
		if tok == token.EOF {
			return nil, errors.New("unable to find method definition")
		}
		if tok == token.LBRACE {
			break
		}
	}

	methods := []*method{}

	for {
		_, tok, lit := p.s.Scan()
		if tok == token.EOF {
			return nil, errors.New("unable to find method definition")
		}
		if tok == token.RBRACE {
			break
		}
		if tok == token.IDENT {
			methodName := lit
			m, err := p.lookForMethod(methodName)
			if err != nil {
				return nil, err
			}
			methods = append(methods, m)
		}
	}
	return methods, nil
}

func (p *parser) lookForMethod(methodName string) (*method, error) {
	collect := []string{}

	for {
		_, tok, lit := p.s.Scan()
		switch tok {
		case token.COMMA,
			token.LPAREN,
			token.RPAREN,
			token.LBRACK,
			token.RBRACK,
			token.LBRACE,
			token.RBRACE,
			token.PERIOD,
			token.ELLIPSIS:
			collect = append(collect, tok.String())
		case token.EOF:
			return nil, fmt.Errorf("unable to parse method definition for '%s'", methodName)
		}

		if tok == token.SEMICOLON {
			break
		}

		if lit != "" {
			collect = append(collect, lit)
		}
	}

	// Joining with a " " in so that the tokeniser can differentiate between
	// "a string" and "astring". Granted, this isn't going to be nice to read,
	// but the idea is to run gofmt (or goimports) later down the line, to
	// enforce standard go syntax.
	return &method{name: methodName, signature: strings.Join(collect, " ")}, nil
}

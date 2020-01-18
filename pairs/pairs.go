/* package pairs allows detecting broken key/value pairs.

A key is defined as a string and a value can be anything.

A missing value is an error:

	logger.Log("name", "frew", "job", "engineer", "age") // missing value

A non-string is also an error:
                                     		// missing key
	logger.Log("message", "successful!",                    3)

The sole analyzer (from NewAnalyzer) in this package takes a -pair-flag
flag that can define any number of the following:

1. Any method named Log, start pairs at 0:

	-pair-func .Log=0

2. The Wrap func from the go.zr.org/common/go/errors package, start pairs at 2:

	-pair-func go.zr.org/common/go/errors.Wrap=2

3. The AddPairs method from the go.zr.org/common/go/errors/details.Pairs type,
start pairs at 0:

	-pair-func go.zr.org/common/go/errors/details.Pairs.AddPairs=0

The other flag defined by this package is -assume-pair flag, which users can
use to define type "safe" for passing around.  The idea is that you'd define
all methods on the type as pair funcs; this means you are passing around the
value instead of a raw slice of interfaces, which could get modified in
surprising ways by users.
*/
package pairs

import (
	"errors"
	"flag"
	"go/ast"
	"go/constant"
	"go/types"
	"regexp"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/astutil"
)

type funcSelector struct{ pkg, typ, fun string }

type funcOffset map[funcSelector]int

var funcOffsetMatcher = regexp.MustCompile(`^(?:(.*?)(?:\.([^\./]+))?)?\.([^\.]+)=(\d+)$`)

func (o funcOffset) Set(v string) error {
	m := funcOffsetMatcher.FindStringSubmatch(v)
	if len(m) != 5 {
		return errors.New("invalid func offset; should be of form [pkg[.type]].<func>=<offset>")
	}
	val, err := strconv.Atoi(m[4])
	if err != nil {
		return err
	}

	o[funcSelector{pkg: m[1], typ: m[2], fun: m[3]}] = val
	return nil
}

func (o funcOffset) String() string {
	return "Woo"
}

type whitelistableType struct{ pkg, typ string }

type typeWhitelist map[whitelistableType]bool

var typeWhitelistMatcher = regexp.MustCompile(`^(.*?)\.([^\./]+)$`)

func (w typeWhitelist) Set(v string) error {
	m := typeWhitelistMatcher.FindStringSubmatch(v)
	if len(m) != 3 {
		return errors.New("invalid type whitelist; should be of form <pkg>.<type>")
	}

	w[whitelistableType{pkg: m[1], typ: m[2]}] = true
	return nil
}

func (w typeWhitelist) String() string {
	return "Woo"
}

// NewAnalyzer returns a fresh pairs analyzer.
func NewAnalyzer() *analysis.Analyzer {
	fset := flag.NewFlagSet("pairs", flag.ContinueOnError)

	offsets := funcOffset{}
	whitelistedTypes := typeWhitelist{}

	fset.Var(offsets, "pair-func", "validate this func")
	fset.Var(whitelistedTypes, "assume-pair", "assume this type is safe")

	// Same comment as on argsCorrect below. --fREW 2020-01-18
	isWhitelisted := func(p *analysis.Pass, e ast.Expr) bool {
		typ := p.TypesInfo.Types[e]
		if p, ok := typ.Type.(*types.Pointer); ok {
			if named, ok := p.Elem().(*types.Named); ok {
				if whitelistedTypes[whitelistableType{pkg: named.Obj().Pkg().Path(), typ: named.Obj().Name()}] {
					return true
				}
			}
		}
		if named, ok := typ.Type.(*types.Named); ok {
			if whitelistedTypes[whitelistableType{pkg: named.Obj().Pkg().Path(), typ: named.Obj().Name()}] {
				return true
			}
		}
		return false

	}

	// it'd be better to make a value that has an argsCorrect method than
	// this weird closure oriented style.  If I get around to it I'll
	// change this. --fREW 2020-01-17
	argsCorrect := func(p *analysis.Pass, name string, offset int, c *ast.CallExpr) {
		if len(c.Args) <= offset {
			return
		}

		// if we only have 1 arg it needs to be one of the whitelisted
		// types
		if len(c.Args)-offset == 1 {
			if isWhitelisted(p, c.Args[offset]) {
				return
			}
		}

		if (len(c.Args)-offset)%2 != 0 {
			p.Reportf(c.Pos(), "%d args passed to %s; must be even", len(c.Args), name)
			return
		}

		for i, a := range c.Args[offset:] {
			if isWhitelisted(p, a) {
				p.Reportf(c.Pos(), "arg %d to %s is a whitelisted type; should pass one or none", i+offset, name)
				return
			}
		}

		for i, a := range c.Args[offset:] {
			if i%2 != 0 {
				continue
			}

			typ := p.TypesInfo.Types[a]

			// TODO prefer *anonymous* constant

			// it's a string constant, this is preferred
			if typ.Value != nil { // constant
				if typ.Value.Kind() != constant.String {
					p.Reportf(a.Pos(), "arg %d to %s is constant %s but should be a constant string",
						i+offset,
						name,
						types.TypeString(typ.Type, nil),
					)
				}
				continue
			}

			if typ.Type != nil { // expression
				b, ok := typ.Type.Underlying().(*types.Basic)
				if ok && b.Kind() == types.String {
					// it's a string expression, this is not preferred, but is acceptable
					continue
				}
				p.Reportf(a.Pos(), "arg %d to %s is expression %s but should be a constant string",
					i+offset,
					name,
					types.TypeString(typ.Type, nil),
				)
			}
		}
	}

	return &analysis.Analyzer{
		Name:  "pairs",
		Doc:   "pairs allows verification of key/value pairs in ...interface{} args; see -pair-func especially",
		Flags: *fset,
		Run: func(p *analysis.Pass) (interface{}, error) {
			i := p.TypesInfo

			for _, f := range p.Files {
				astutil.Apply(f, func(cur *astutil.Cursor) bool {
					c, ok := cur.Node().(*ast.CallExpr)
					if !ok {
						return true
					}
					s, ok := c.Fun.(*ast.SelectorExpr) // possibly method calls
					if !ok {
						return true
					}

					// package functions
					nv, ok := i.Selections[s]
					if !ok {
						pkgName := i.Uses[s.X.(*ast.Ident)].(*types.PkgName) // 😅
						path := pkgName.Imported().Path()

						offset, ok := offsets[funcSelector{pkg: path, fun: s.Sel.Name}]
						if !ok { // we don't care about this function
							return true
						}

						argsCorrect(p, path+"."+s.Sel.Name, offset, c)

						return true
					}

					named, ok := nv.Recv().(*types.Named)
					if !ok {
						// if there is no receiver (or
						// it's anonymous) it's some
						// weird thing like an
						// anonymous struct with a func
						// being called.  structs with func
						// fields do not conform to interfaces,
						// and thus are not relevant to this
						return true
					}

					// try generous interface first
					offset, ok := offsets[funcSelector{fun: s.Sel.Name}]
					if !ok {
						// otherwise try concrete type
						offset, ok = offsets[funcSelector{fun: s.Sel.Name, pkg: named.Obj().Pkg().Path(), typ: named.Obj().Name()}]
					}
					if !ok {
						return true
					}

					argsCorrect(p, types.SelectionString(nv, nil), offset, c)
					return true
				}, nil)
			}
			return nil, nil
		},
	}
}

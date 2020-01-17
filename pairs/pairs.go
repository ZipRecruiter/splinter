/* package pairs allows detecting broken key/value pairs.

A key is defined as a string and a value can be anything.

A missing value is an error:

	logger.Log("name", "frew", "job", "engineer", "age") // missing value

A non-string is also an error:
                                     		// missing key
	logger.Log("message", "successful!",                    3)

The sole analyzer (from NewAnalyzer) in this package takes a -pair-flag
func that can define any number of the following:

1. Any method named Log, start pairs at 0:

	-pair-func .Log=0

2. The Wrap func from the go.zr.org/common/go/errors package, start pairs at 2:

	-pair-func go.zr.org/common/go/errors.Wrap=2

3. The AddPairs method from the go.zr.org/common/go/errors/details.Pairs type,
start pairs at 0:

	-pair-func go.zr.org/common/go/errors/details.Pairs.AddPairs=0
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

type funcSelector struct {
	pkg, typ, fun string
}

type funcOffset map[funcSelector]int

var funcOffsetMatcher = regexp.MustCompile(`^(?:(.*?)(?:\.([^\./]+))?)?\.([^\.]+)=(\d+)$`)

func (o funcOffset) Set(v string) error {
	m := funcOffsetMatcher.FindStringSubmatch(v)
	if len(m) != 5 {
		return errors.New("invalis func offset; should be of form [pkg[.type]].<func>=<offset>")
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

// NewAnalyzer returns a fresh pairs analyzer.
func NewAnalyzer() *analysis.Analyzer {
	fset := flag.NewFlagSet("pairs", flag.ContinueOnError)

	offsets := funcOffset{}
	fset.Var(offsets, "pair-func", "validate this func")
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

						if (len(c.Args)-offset)%2 != 0 {
							p.Reportf(c.Pos(), "%d args passed to %s; must be even", len(c.Args), path+"."+s.Sel.Name)
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

					if len(c.Args)%2 != 0 {
						p.Reportf(c.Pos(), "%d args passed to %s; must be even", len(c.Args), types.SelectionString(nv, nil))
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

func argsCorrect(p *analysis.Pass, name string, offset int, c *ast.CallExpr) bool {
	if len(c.Args) <= offset {
		return true
	}

	ret := true

	// XXX assume all good if sole arg is a whitelisted type
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
				ret = false
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
			ret = false
		}
	}

	return ret
}

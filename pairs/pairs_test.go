package pairs

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalysis(t *testing.T) {
	filemap := map[string]string{
		"a/a.go": `package a

import "a/b"

type logger int

func (l logger) Log(inputs ...interface{}) { }

func Foo() {
	p := b.NewPairs("foo", 1)
	p.AddPairs("bar", 2)

	l := logger(0)
	// generous interface
	l.Log("foo", "bar", "baz") // want "3 args passed to method \\(a.logger\\) Log\\(inputs ...interface{}\\); must be even"
	l.Log(1, "bar") // want "arg 0 to method \\(a.logger\\) Log\\(inputs ...interface{}\\) is constant int but should be a constant string"
	l.Log(l, "bar") // want "arg 0 to method \\(a.logger\\) Log\\(inputs ...interface{}\\) is expression a.logger but should be a constant string"
	l.Log("foo", 1, "bar", "baz")
	l.Log(p)

	// concrete method
	y := b.Y(0)
	y.Z("frew") // want "1 args passed to method \\(a/b.Y\\) Z\\(inputs ...interface{}\\); must be even"
	y.Z(1, "frew") // want "arg 0 to method \\(a/b.Y\\) Z\\(inputs ...interface{}\\) is constant int but should be a constant string"
	y.Z(l, "frew") // want "arg 0 to method \\(a/b.Y\\) Z\\(inputs ...interface{}\\) is expression a.logger but should be a constant string"
	y.Z("foo", "frew")
	y.Z(p)

	// package func
	b.X("", "frew") // want "2 args passed to a/b.X; must be even"
	b.X("", 1, "frew") // want "arg 1 to a/b.X is constant int but should be a constant string"
	b.X("", l, "frew") // want "arg 1 to a/b.X is expression a.logger but should be a constant string"
	b.X("", "foo", "frew")

	// variable key is ok for now
	x := "station"
	b.X("", x, "frew")

	b.X("woo", p) // this is ok because we've defined b.Pairs as "safe"

	b.X("foo", "key/vales", p) // want "arg 2 to a/b.X is a whitelisted type; should pass one or none"
}

`,
		"a/b/b.go": `package b

func X(a string, inputs ...interface{}) {}

type Y int

func (y Y) Z(inputs ...interface{}) {}

type Pairs struct {
	values []interface{}
}

func NewPairs(p ...interface{}) *Pairs { return &Pairs{p} }

func (p *Pairs) AddPairs(i ...interface{}) {
	p.values = append(p.values, i...)
}
`,
	}

	dir, cleanup, err := analysistest.WriteFiles(filemap)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	a := NewAnalyzer()
	// generous interface
	if err := a.Flags.Set("pair-func", ".Log=0"); err != nil {
		t.Fatal(err)
	}

	// concrete type
	if err := a.Flags.Set("pair-func", "a/b.Y.Z=0"); err != nil {
		t.Fatal(err)
	}

	// func
	if err := a.Flags.Set("pair-func", "a/b.X=1"); err != nil {
		t.Fatal(err)
	}

	if err := a.Flags.Set("pair-func", "a/b.NewPairs=0"); err != nil {
		t.Fatal(err)
	}
	if err := a.Flags.Set("pair-func", "a/b.Pairs.AddPairs=0"); err != nil {
		t.Fatal(err)
	}

	if err := a.Flags.Set("assume-pair", "a/b.Pairs"); err != nil {
		t.Fatal(err)
	}

	analysistest.Run(t, dir, a, "a")
}

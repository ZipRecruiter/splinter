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
	l := logger(0)
	l.Log("foo", "bar", "baz") // want "3 args passed to .*; must be even"
	l.Log(1, "bar") // want "arg 0 to type .* is constant int but should be a constant string"
	l.Log("foo", 1, "bar", "baz")

	b.X("", 1, "frew") // want "arg 1 to a/b.X is constant int but should be a constant string"
	b.X("", "frew") // want "2 args passed to a/b.X; must be even"
}

`,
		"a/b/b.go": `package b

func X(a string, inputs ...interface{}) {

}
`,
	}

	dir, cleanup, err := analysistest.WriteFiles(filemap)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	a := NewAnalyzer()
	if err := a.Flags.Set("pair-func", ".Log=0"); err != nil {
		t.Fatal(err)
	}
	if err := a.Flags.Set("pair-func", "a/b.X=1"); err != nil {
		t.Fatal(err)
	}
	analysistest.Run(t, dir, a, "a")
}

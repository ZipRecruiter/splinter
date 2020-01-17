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
	// generous interface
	l.Log("foo", "bar", "baz") // want "3 args passed to method \\(a.logger\\) Log\\(inputs ...interface{}\\); must be even"
	l.Log(1, "bar") // want "arg 0 to method \\(a.logger\\) Log\\(inputs ...interface{}\\) is constant int but should be a constant string"
	l.Log("foo", 1, "bar", "baz")

	// package func
	b.X("", 1, "frew") // want "arg 1 to a/b.X is constant int but should be a constant string"
	b.X("", "frew") // want "2 args passed to a/b.X; must be even"

	// concrete method
	y := b.Y(0)
	y.Z(1, "frew") // want "arg 0 to method \\(a/b.Y\\) Z\\(inputs ...interface{}\\) is constant int but should be a constant string"
	y.Z("frew") // want "1 args passed to method \\(a/b.Y\\) Z\\(inputs ...interface{}\\); must be even"
}

`,
		"a/b/b.go": `package b

func X(a string, inputs ...interface{}) {}

type Y int

func (y Y) Z(inputs ...interface{}) {}
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
	analysistest.Run(t, dir, a, "a")
}

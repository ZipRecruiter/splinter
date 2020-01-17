package pairs

import (
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFuncOffset(t *testing.T) {
	type Test struct {
		in  string
		out funcOffset
	}

	tests := []Test{
		{".Log=0", funcOffset{funcSelector{fun: "Log"}: 0}},
		{"go.zr.org/common/go/errors/details.Pairs.AddPairs=1", funcOffset{funcSelector{pkg: "go.zr.org/common/go/errors/details", typ: "Pairs", fun: "AddPairs"}: 1}},
		{"go.zr.org/common/go/errors.Wrap=2", funcOffset{funcSelector{pkg: "go.zr.org/common/go/errors", fun: "Wrap"}: 2}},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			f := funcOffset{}
			if err := f.Set(test.in); err != nil {
				panic(err)
			}
			if d := cmp.Diff(test.out, f); d != "" {
				t.Errorf("unexpected result (-expected +got):\n%s", d)
			}
		})
	}
}

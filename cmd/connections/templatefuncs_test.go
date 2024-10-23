package connections

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTplInlineArray(t *testing.T) {
	strSlice := []string{"a", "b", "c"}
	assert.Equal(t, "[a,b,c]", TplInlineArray(strSlice))

	intSlice := []int{1, 2, 3}
	assert.Equal(t, "[1,2,3]", TplInlineArray(intSlice))

	strIntSlice := []any{"a", 1, "b", 2}
	assert.Equal(t, "[a,1,b,2]", TplInlineArray(strIntSlice))
}

func TestTplToYaml(t *testing.T) {
	expectedWithoutIndents := `a: 1
b:
  - 1
  - 2
  - 3
`
	expectedWithIndents := `  a: 1
  b:
    c: 2
    d: 3
`
	obj1 := struct {
		A int
		B []int
	}{
		A: 1,
		B: []int{1, 2, 3},
	}

	obj2 := struct {
		A int
		B any
	}{
		A: 1,
		B: struct {
			C int
			D int
		}{
			C: 2,
			D: 3,
		},
	}

	assert.Equal(t, expectedWithoutIndents, TplToYaml(obj1, 2, 0))
	assert.Equal(t, expectedWithIndents, TplToYaml(obj2, 2, 1))
}

package connections

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTplValueToYaml(t *testing.T) {
	assert.Equal(t, "\"y\"", TplValueToYaml("y"))
	assert.Equal(t, "a", TplValueToYaml("a"))
	assert.Equal(t, "5", TplValueToYaml(5))
	assert.Equal(t, "2.22", TplValueToYaml(2.22))
	assert.Equal(t, "true", TplValueToYaml(true))
}

func TestTplInlineArray(t *testing.T) {
	strSlice := []string{"a", "b", "c", "y"}
	assert.Equal(t, "[a, b, c, \"y\"]", TplInlineArray(strSlice))

	intSlice := []int{1, 2, 3}
	assert.Equal(t, "[1, 2, 3]", TplInlineArray(intSlice))

	strIntSlice := []any{"a", 1, "b", 2}
	assert.Equal(t, "[a, 1, b, 2]", TplInlineArray(strIntSlice))
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

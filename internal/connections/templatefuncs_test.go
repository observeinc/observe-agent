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

func TestTplToJSON(t *testing.T) {
	testData := map[string]any{
		"a": 1,
		"b": []int{1, 2, 3},
		"c": "quote' \"",
	}
	assert.Equal(t, `{"a":1,"b":[1,2,3],"c":"quote' \""}`, TplToJSON(testData))
}

func TestTplJoin(t *testing.T) {
	assert.Equal(t, "a, b, c", TplJoin(", ", []string{"a", "b", "c"}))
	assert.Equal(t, "12", TplJoin("", []any{"1", 2}))
	assert.Equal(t, "single", TplJoin("not used", []any{"single"}))
}

func TestTplFlatten(t *testing.T) {
	assert.Equal(t, []any{"a", "b", "c", 1, 2, 3}, TplFlatten(", ", []string{"a", "b", "c"}, []int{1, 2, 3}))
	assert.Equal(t, []any{"a", "b", "c", "d", 123}, TplFlatten(", ", []any{"a", "b", []any{"c", "d", 123}}))
}

func TestTplConcat(t *testing.T) {
	assert.Equal(t, []any{"a", "b", "c", 1, 2, 3}, TplConcat([]string{"a", "b", "c"}, []int{1, 2, 3}))
	assert.Equal(t, []any{"a", "b", "c", "d", "e", 123}, TplConcat([]string{"a", "b", "c"}, "d", "e", 123))
}

func TestTplList(t *testing.T) {
	assert.Equal(t, []any{"a", "b", "c"}, TplList("a", "b", "c"))
	assert.Equal(t, []any{1}, TplList(1))
	assert.Equal(t, []any{[]any{1, 2}}, TplList([]any{1, 2}))
}

func TestTplUniq(t *testing.T) {
	assert.Equal(t, []any{"a", 1, "b", 2}, TplUniq([]any{"a", "a", 1, "b", 2, "a", 1, "b", 2, 1}))
}

func TestTplDict(t *testing.T) {
	assert.Equal(t, map[string]any{}, TplDict())
	assert.Equal(t, map[string]any{"a": 1, "b": 2}, TplDict("a", 1, "b", 2))
	assert.Equal(t, map[string]any{"a": 1, "b": 2, "c": map[string]any{"d": 3}}, TplDict("a", 1, "b", 2, "c", TplDict("d", 3)))
	assert.Equal(t, map[string]any{"1": 1}, TplDict(1, 1))
}

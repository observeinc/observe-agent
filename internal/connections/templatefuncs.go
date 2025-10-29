package connections

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

var TemplateFuncMap = template.FuncMap{
	"inlineArrayInt": TplInlineArray[int],
	"inlineArrayStr": TplInlineArray[string],
	"valToYaml":      TplValueToYaml,
	"objToYaml":      TplToYaml,
	"join":           TplJoin,
	"flatten":        TplFlatten,
	"concat":         TplConcat,
	"list":           TplList,
	"dict":           TplDict,
	"uniq":           TplUniq,
	"json":           TplToJSON,
	"getenv":         os.Getenv,
	"add": func(values ...int) int {
		sum := 0
		for _, i := range values {
			sum += i
		}
		return sum
	},
}

func TplInlineArray[T any](arr []T) string {
	strs := make([]string, len(arr))
	for i := range arr {
		strs[i] = TplValueToYaml(arr[i])
	}
	return "[" + strings.Join(strs, ", ") + "]"
}

func TplValueToYaml(value any) string {
	b, err := yaml.Marshal(value)
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(b))
}

func TplToYaml(data any, tabSize int, numTabs int) string {
	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(tabSize)
	err := encoder.Encode(data)
	if err != nil {
		panic(err)
	}
	yamlStr := b.String()
	if numTabs > 0 && len(yamlStr) > 0 {
		var indentStr = strings.Repeat(" ", numTabs*tabSize)
		yamlStr = indentStr + strings.ReplaceAll(yamlStr, "\n", "\n"+indentStr)
		yamlStr, _ = strings.CutSuffix(yamlStr, indentStr)
	}
	return yamlStr
}

func TplToJSON(data any) string {
	strB, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return string(strB)
}

func TplJoin(sep string, values any) string {
	if strs, ok := values.([]string); ok {
		return strings.Join(strs, sep)
	}
	valuesArr, ok := values.([]any)
	if !ok {
		return ""
	}
	strs := make([]string, len(valuesArr))
	for i, v := range valuesArr {
		if str, ok := v.(string); ok {
			// If the value is a string, use it directly.
			strs[i] = str
		} else {
			// Otherwise convert via yaml.
			strs[i] = TplValueToYaml(v)
		}
	}
	return strings.Join(strs, sep)
}

func TplFlatten(sep string, values ...any) []any {
	result := make([]any, 0, len(values))
	var appendValues func(reflect.Value)
	appendValues = func(v reflect.Value) {
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			for i := 0; i < v.Len(); i++ {
				appendValues(reflect.ValueOf(v.Index(i).Interface()))
			}
		} else {
			result = append(result, v.Interface())
		}
	}
	appendValues(reflect.ValueOf(values))
	return result
}

func TplConcat(values ...any) []any {
	result := make([]any, 0, len(values))
	for _, v := range values {
		t := reflect.TypeOf(v)
		if t.Kind() == reflect.Slice {
			reflectVal := reflect.ValueOf(v)
			for i := 0; i < reflectVal.Len(); i++ {
				result = append(result, reflectVal.Index(i).Interface())
			}
			continue
		}
		// If the value is not a slice, append it to the concatenation result
		result = append(result, v)
	}
	return result
}

func TplList(values ...any) []any {
	return values
}

func TplUniq(values []any) []any {
	seen := make(map[any]struct{})
	result := make([]any, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	return result
}

func keyToString(key any) string {
	if str, ok := key.(string); ok {
		return str
	}
	return TplValueToYaml(key)
}

func TplDict(values ...any) map[string]any {
	dict := map[string]any{}
	n := len(values)
	if n%2 != 0 {
		panic("dict: odd number of arguments")
	}
	for i := 0; i < n; i += 2 {
		dict[keyToString(values[i])] = values[i+1]
	}
	return dict
}

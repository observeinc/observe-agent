package connections

import (
	"bytes"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

var TemplateFuncMap = template.FuncMap{
	"inlineArrayInt": TplInlineArray[int],
	"inlineArrayStr": TplInlineArray[string],
	"valToYaml":      TplValueToYaml,
	"objToYaml":      TplToYaml,
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

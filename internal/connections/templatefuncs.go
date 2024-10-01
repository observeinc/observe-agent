package connections

import (
	"bytes"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

func GetTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"inlineArrayInt": TplInlineArray[int],
		"inlineArrayStr": TplInlineArray[string],
		"valToYaml":      TmplValueToYaml,
		"objToYaml":      TplToYaml,
	}
}

func TplInlineArray[T any](arr []T) string {
	strs := make([]string, len(arr))
	for i := range arr {
		strs[i] = TmplValueToYaml(arr[i])
	}
	return "[" + strings.Join(strs, ",") + "]"
}

func TmplValueToYaml(value any) string {
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

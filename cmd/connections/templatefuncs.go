package connections

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

func GetTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"inlineArrayInt": TplInlineArray[int],
		"inlineArrayStr": TplInlineArray[string],
		"objToYaml":      TplToYaml,
	}
}

func TplInlineArray[T any](arr []T) string {
	strs := make([]string, len(arr))
	for i := range arr {
		strs[i] = fmt.Sprintf("%v", arr[i])
	}
	return "[" + strings.Join(strs, ",") + "]"
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

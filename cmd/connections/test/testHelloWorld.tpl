string: {{ .TestStr }}
in-line-array: {{ inlineArrayInt .TestArr1 }}
multi-line-array:
{{- range $i, $item := .TestArr2 }}
  - {{ $item }}
{{- end }}
nested-obj:
{{ objToYaml .TestObj 2 1 }}
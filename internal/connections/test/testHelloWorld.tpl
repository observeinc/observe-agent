string: {{ .TestStr }}
in-line-array: [{{ join ", " .TestArr1 }}]
multi-line-array:
{{- range $i, $item := .TestArr2 }}
  - {{ $item }}
{{- end }}
nested-obj:
{{- mustToYaml .TestObj | nindent 2 }}

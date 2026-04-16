# Help for ~help~

The ~help~ command looks up guidance on a command.

## Usage:

  ~help say~
  Find help on the command ~say~.

Here are some *skills* and *commands* to look up to get started:

{{ if gt (len .Commands) 0 -}}
Commands:
{{ range $category, $commandList := .Commands -}}
{{ if ne $category "" }}
**{{ uc $category }}**
{{ end -}}
{{ range $cmdInfo := $commandList -}}
  - ~{{ $cmdInfo.Command }}~{{ if $cmdInfo.Missing }} ***{{ end }}
{{ end }}
{{ end }}
{{ end }}

{{- if gt (len .Skills) 0 -}}
Skills:
{{- range $category, $commandList := .Skills -}}
{{ if ne $category "" }}
**{{ uc $category }}**
{{ end -}}
{{ range $cmdInfo := $commandList -}}
  - ~{{ $cmdInfo.Command }}~{{ if $cmdInfo.Missing }} ***{{ end }}
{{ end }}
{{ end }}
{{ end }}

{{- if gt (len .Admin) 0 -}}
Admin:
{{- range $category, $commandList := .Admin -}}
{{ if ne $category "" }}
**{{ uc $category }}**
{{ end -}}
{{ range $cmdInfo := $commandList -}}
  - ~{{ $cmdInfo.Command }}~{{ if $cmdInfo.Missing }} ***{{ end }}
{{ end }}
{{ end }}
{{ end }}

**See also:** ~help gomud~

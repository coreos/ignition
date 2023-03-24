---
title: Config Spec v{{ .Version }}
parent: Configuration specifications
nav_order: {{ .NavOrder }}
---

# Configuration Specification v{{.Version}}

{{ if .Version.PreRelease -}}
_NOTE_: This pre-release version of the specification is experimental and is subject to change without notice or regard to backward compatibility.

{{ end -}}
The Ignition configuration is a JSON document conforming to the following specification, with **_italicized_** entries being optional:


// Spike module. Deliberately separate from the main module so that
// throwaway benchmark dependencies never enter Granite's dependency graph
// (see docs/adr/README.md).
module github.com/CyberSecDef/granite/spikes

go 1.23

require (
	github.com/goccy/go-yaml v1.19.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

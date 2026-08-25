// examples/configservice/schema_embed.go

package configservice

import _ "embed"

//go:generate cue exp gengotypes ./...

// definition is the CUE schema consumed by s99config. The Go types decoded
// from it live in cue_types_config_gen.go.
//
//go:embed config.cue
var definition []byte

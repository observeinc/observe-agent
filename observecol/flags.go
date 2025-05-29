package observecol

import (
	"github.com/spf13/pflag"
)

// Flag name and description copied directly from otel collector
const (
	configFlag            = "config"
	configFlagDescription = "OpenTelemetry config file(s), note that only a single location can be set per flag entry e.g. " +
		"`--config=file:/path/to/first --config=file:path/to/second`."
	setFlag            = "set"
	setFlagDescription = "Set arbitrary OpenTelemetry component config properties. The component has to be defined in " +
		"the bundled or provided OpenTelemetry config files and this flag has a higher precedence. " +
		"Array config properties are overridden and maps are joined. Example --set=processors.batch.timeout=2s"
)

var otelConfigs []string
var otelSets []string

func AddConfigFlags(flags *pflag.FlagSet) {
	flags.StringSliceVar(&otelConfigs, configFlag, []string{}, configFlagDescription)
	flags.StringSliceVar(&otelSets, setFlag, []string{}, setFlagDescription)
}

package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	kongtoml "github.com/alecthomas/kong-toml"
	kongyaml "github.com/alecthomas/kong-yaml"
)

func main() {
	cli := CLI{
		Plugins: kong.Plugins{
			&ListCmd{},
			&RebindCmd{},
			&VersionCmd{},
		},
	}

	options := []kong.Option{
		kong.Bind(&cli),
		kong.Name("auto-vfio"),
		kong.Description("Automate the setup of VFIO passthrough on Linux systems"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: false,
		}),
		kong.Vars{
			"supported_formats": strings.Join(supportedConfigFormats, ", "),
			"log_levels":        strings.Join(LogLevels, ", "),
			"default_log_level": DefaultLogLevel.String(),
		},
	}

	configFile := cli.Globals.ConfigFile.String()
	if f, err := os.Stat(configFile); err == nil && !f.IsDir() {
		switch filepath.Ext(configFile) {
		case ".json":
			options = append(options, kong.Configuration(kong.JSON, configFile))
		case ".yaml", ".yml":
			options = append(options, kong.Configuration(kongyaml.Loader, configFile))
		case ".toml":
			options = append(options, kong.Configuration(kongtoml.Loader, configFile))
		case "":
		default:
			cli.config.Logger().Fatal().Msgf("Unsupported config file format %q", configFile)
		}
	}

	ctx := kong.Parse(&cli, options...)

	var err error
	cli.config, err = NewConfig(
		WithLogLevel(cli.LogLevel),
	)
	if err != nil {
		cli.config.Logger().Fatal().Err(err).
			Msg("Failed to create config")
	}

	err = ctx.Run(&cli.Globals)
	if err != nil {
		cli.config.Logger().Fatal().Err(err).
			Msgf("Failed to run command %q", ctx.Command())
	}
}

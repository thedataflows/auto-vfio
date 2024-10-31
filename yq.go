package main

import (
	"bufio"
	"bytes"
	"container/list"
	"fmt"
	"os"

	"github.com/mikefarah/yq/v4/pkg/yqlib"
	logging "gopkg.in/op/go-logging.v1"
)

// yq evaluates a yq expression on the given data and returns the result
func yq(globals *Globals, expression string, data []byte) (*list.List, error) {
	// Set up yq logging
	backend := logging.AddModuleLevel(
		logging.NewBackendFormatter(
			logging.NewLogBackend(os.Stderr, "", 0),
			logging.MustStringFormatter("%{time:15:04:05} %{level:.3s} [yq] %{shortfile} %{message}"),
		),
	)

	switch globals.config.Logger().GetLevel().String() {
	case "debug":
		backend.SetLevel(logging.DEBUG, "")
	case "info":
		backend.SetLevel(logging.INFO, "")
	case "warning":
		backend.SetLevel(logging.WARNING, "")
	case "error":
		backend.SetLevel(logging.ERROR, "")
	default:
		backend.SetLevel(logging.CRITICAL, "")
	}
	logging.SetBackend(backend)

	// Parse data
	decoder := yqlib.NewJSONDecoder()
	err := decoder.Init(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("error initializing JSON decoder: %w", err)
	}
	node, err := decoder.Decode()
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}

	// Evaluate expression and filter nodes
	if expression == "" {
		expression = "."
	}
	matchingNodes, err := yqlib.NewAllAtOnceEvaluator().
		EvaluateNodes(expression, node)
	if err != nil {
		return nil, fmt.Errorf("error evaluating nodes: %w", err)
	}

	return matchingNodes, nil
}

func yqEncode(matchingNodes *list.List, outFormat string, outColor bool) ([]byte, error) {
	// Encode results
	if len(outFormat) == 0 {
		outFormat = "yaml"
	}
	var encoder yqlib.Encoder
	switch outFormat {
	case "yaml":
		encoder = yqlib.NewYamlEncoder(yqlib.YamlPreferences{
			Indent:                      2,
			ColorsEnabled:               outColor,
			LeadingContentPreProcessing: true,
			PrintDocSeparators:          true,
			UnwrapScalar:                true,
			EvaluateTogether:            false,
		})
	case "xml":
		encoder = yqlib.NewXMLEncoder(yqlib.NewDefaultXmlPreferences())
	case "json":
		encoder = yqlib.NewJSONEncoder(
			yqlib.JsonPreferences{
				Indent:        2,
				ColorsEnabled: outColor,
				UnwrapScalar:  false,
			})
	case "toml":
		encoder = yqlib.NewTomlEncoder()
	case "props":
		encoder = yqlib.NewPropertiesEncoder(yqlib.ConfiguredPropertiesPreferences)
	case "shell":
		encoder = yqlib.NewShellVariablesEncoder()
	case "csv":
		encoder = yqlib.NewCsvEncoder(yqlib.ConfiguredCsvPreferences)
	case "tsv":
		encoder = yqlib.NewCsvEncoder(yqlib.ConfiguredTsvPreferences)
	default:
		return nil, fmt.Errorf("unsupported output format: %s", outFormat)
	}

	var output bytes.Buffer
	err := yqlib.NewPrinter(
		encoder,
		yqlib.NewSinglePrinterWriter(bufio.NewWriter(&output)),
	).PrintResults(matchingNodes)
	if err != nil {
		return nil, fmt.Errorf("error printing results: %w", err)
	}

	return output.Bytes(), nil
}

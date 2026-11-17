package config

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"go.yaml.in/yaml/v3"
)

type fileValues struct {
	Listen     *string
	ProjectDir *string
}

func readConfigFile(path string, required bool) (fileValues, error) {
	data, err := os.ReadFile(path)
	switch {
	case os.IsNotExist(err) && !required:
		return fileValues{}, nil
	case err != nil:
		return fileValues{}, fmt.Errorf("config file %q: %w", path, err)
	}
	values, err := decodeYAML(data)
	if err != nil {
		return fileValues{}, fmt.Errorf("config file %q: %w", path, err)
	}
	return values, nil
}

// Inspect node types because YAML's typed string decoding also accepts numbers
// and booleans. Overrides must not hide malformed configuration.
func decodeYAML(data []byte) (fileValues, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		if err == io.EOF {
			return fileValues{}, nil
		}
		return fileValues{}, err
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return fileValues{}, fmt.Errorf("expected a YAML mapping")
	}
	root := document.Content[0]
	var values fileValues
	for i := 0; i < len(root.Content); i += 2 {
		key, value := root.Content[i], root.Content[i+1]
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
			return fileValues{}, fmt.Errorf("setting name must be a string")
		}
		var target **string
		switch key.Value {
		case "listen":
			target = &values.Listen
		case "project-dir":
			target = &values.ProjectDir
		default:
			return fileValues{}, fmt.Errorf("unknown setting %q", key.Value)
		}
		if *target != nil {
			return fileValues{}, fmt.Errorf("duplicate setting %q", key.Value)
		}
		if value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
			return fileValues{}, fmt.Errorf("%s: expected a string value", key.Value)
		}
		*target = &value.Value
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return fileValues{}, err
		}
		return fileValues{}, fmt.Errorf("expected one YAML document")
	}
	return values, nil
}

//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package prompt

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	YAML_INCLUDE = "include"
	YAML_PROMPT  = "prompt"
	YAML_INPUT   = "input"
	YAML_BLOB    = "blob"
	YAML_FORMAT  = "format"

	YAML_REGISTRY            = "registry"
	YAML_REGISTRY_NAME       = "name"
	YAML_REGISTRY_TYPE       = "type"
	YAML_REGISTRY_ABOUT      = "about"
	YAML_REGISTRY_SYNTAX     = "syntax"
	YAML_REGISTRY_PROPERTIES = "properties"
)

// Parse prompt file
func ParseFile(path string) (*viper.Viper, error) {
	if strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") {
		return yamlParseFile(path)
	}

	return textParseFile(path)
}

func yamlParseFile(path string) (*viper.Viper, error) {
	in := viper.New()
	in.SetConfigType("yaml")
	in.SetConfigFile(path)

	err := in.ReadInConfig()
	if err != nil {
		return nil, err
	}

	include := in.GetString(YAML_INCLUDE)
	if len(include) == 0 {
		return in, nil
	}

	c, err := yamlParseFile(filepath.Join(filepath.Dir(path), include))
	if err != nil {
		return nil, err
	}

	for key, val := range c.AllSettings() {
		if !in.IsSet(key) {
			in.Set(key, val)
		}
	}

	return in, nil
}

func textParseFile(path string) (*viper.Viper, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	in := viper.New()
	in.Set(YAML_PROMPT, string(b))
	return in, nil
}

func Parse(r io.Reader) (*viper.Viper, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	in := viper.New()
	in.SetConfigType("yaml")

	err = in.ReadConfig(bytes.NewBuffer(b))
	if err != nil {
		// Unable to parse input as YAML, this is textual prompt
		in.Set(YAML_PROMPT, string(b))
		return in, nil
	}

	include := in.GetString(YAML_INCLUDE)
	if len(include) == 0 {
		return in, nil
	}

	c, err := yamlParseFile(filepath.Join(dir, include))
	if err != nil {
		return nil, err
	}

	for key, val := range c.AllSettings() {
		if !in.IsSet(key) {
			in.Set(key, val)
		}
	}

	return in, nil
}

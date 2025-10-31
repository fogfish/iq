//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package adapter

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/fogfish/iq/internal/core"
	"github.com/fogfish/iq/internal/prompt"
	"github.com/kshard/thinker"
	"github.com/kshard/thinker/command"
	"github.com/spf13/viper"
)

func DecodeViperToPrompt(in *viper.Viper) (*core.Prompt, error) {
	task, err := applyTemplate(
		in.GetString(prompt.YAML_PROMPT),
		in.GetStringMap(prompt.YAML_INPUT),
	)
	if err != nil {
		return nil, err
	}

	var req core.Prompt
	req.Task = task

	if blob := in.GetString(prompt.YAML_BLOB); len(blob) != 0 {
		req.Blob = blob
	}

	if in.GetString(prompt.YAML_FORMAT) == "json" {
		req.Format = core.FORMAT_JSON
	}

	return &req, nil
}

func applyTemplate(t string, c map[string]any) (string, error) {
	txt, err := template.New("").Parse(t)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	err = txt.Execute(&sb, c)
	if err != nil {
		return "", err
	}

	return sb.String(), nil
}

func DecodeViperToRegistry(in *viper.Viper, workdir string) (thinker.Registry, error) {
	r := command.NewRegistry()
	cmds, ok := in.Get(prompt.YAML_REGISTRY).([]any)

	if !ok {
		return nil, fmt.Errorf("command registry is misconfigured")
	}

	for _, cmd := range cmds {
		switch cmd.(type) {
		case string:
			// switch v {
			// case command.BASH:
			// 	r.Register(command.Bash("", workdir))
			// case command.PYTHON:
			// 	r.Register(command.Python(workdir))
			// case command.GOLANG:
			// 	r.Register(command.Golang(workdir))
			// default:
			// 	fmt.Fprintf(os.Stderr, " ‼️ command %s is unknown, define custom command", v)
			// }

		case map[string]any:
			var (
			// err error
			// c          thinker.Cmd
			// properties []any
			)
			// c.Cmd, err = value[string](v, prompt.YAML_REGISTRY_NAME)
			// if err != nil {
			// 	return nil, err
			// }

			// c.About, err = value[string](v, prompt.YAML_REGISTRY_ABOUT)
			// if err != nil {
			// 	return nil, err
			// }

			// c.Syntax, err = value[string](v, prompt.YAML_REGISTRY_SYNTAX)
			// if err != nil {
			// 	return nil, err
			// }

			// properties, err = value[[]any](v, prompt.YAML_REGISTRY_PROPERTIES)
			// if err != nil {
			// 	return nil, err
			// }

			// c.Args = make([]thinker.Arg, len(properties))
			// for _ := range properties {
			// _, ok := prop.(map[string]any)
			// if !ok {
			// 	return nil, fmt.Errorf("command registry is misconfigured, properties of %s has to be a list of objects", c.Cmd)
			// }

			// var arg thinker.Arg
			// arg.Name, err = value[string](kv, prompt.YAML_REGISTRY_NAME)
			// if err != nil {
			// 	return nil, err
			// }

			// arg.Type, err = value[string](kv, prompt.YAML_REGISTRY_TYPE)
			// if err != nil {
			// 	return nil, err
			// }

			// arg.About, err = value[string](kv, prompt.YAML_REGISTRY_ABOUT)
			// if err != nil {
			// 	return nil, err
			// }

			// c.Args[i] = arg
			// }

			// if err = r.Register(command.Cmd(workdir, c)); err != nil {
			// 	return nil, err
			// }
		}
	}

	return r, nil
}

func value[T any](kv map[string]any, key string) (T, error) {
	var nul T

	val, has := kv[key]
	if !has {
		return nul, fmt.Errorf("command registry is misconfigured, missing key %s", key)
	}

	tvl, ok := val.(T)
	if !ok {
		return nul, fmt.Errorf("command registry is misconfigured, key `%s` has wrong type", key)
	}

	return tvl, nil
}

//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package service

import (
	"html/template"
	"strings"

	spec "github.com/fogfish/iq/internal/prompt"
	"github.com/kshard/chatter"
	"github.com/kshard/thinker/prompt/jsonify"
	"github.com/spf13/viper"
)

// Creates LLM prompt
func fromViper(in *viper.Viper) (prompt chatter.Prompt, err error) {
	task := in.GetString(spec.YAML_PROMPT)
	args := in.GetStringMap(spec.YAML_INPUT)

	prompt.Task, err = text(task, args)
	if err != nil {
		return
	}

	if blob := in.GetString(spec.YAML_BLOB); len(blob) != 0 {
		prompt.With(chatter.Blob("Input document", blob))
	}

	if in.GetString(spec.YAML_FORMAT) == "json" {
		jsonify.Strings.Harden(&prompt)
	}

	return
}

func text(t string, c map[string]any) (string, error) {
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

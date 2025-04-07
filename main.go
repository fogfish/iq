//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package main

import (
	"fmt"

	"github.com/fogfish/iq/cmd"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cmd.Execute(fmt.Sprintf("iq/%s (%s), %s", version, commit[:7], date))
}

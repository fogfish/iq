//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package processor

import (
	"maps"

	"github.com/fogfish/iq/internal/iosystem"
)

// copyMetadata creates a shallow copy of metadata.
func copyMetadata(m iosystem.Metadata) iosystem.Metadata {
	copy := iosystem.Metadata{
		ContentType: m.ContentType,
		Extension:   m.Extension,
		Size:        m.Size,
	}
	maps.Copy(copy.Custom, m.Custom)

	return copy
}

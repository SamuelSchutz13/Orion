//go:build !windows

package logs

import "golang.org/x/text/encoding/charmap"

func oemCharmap() *charmap.Charmap { return nil }

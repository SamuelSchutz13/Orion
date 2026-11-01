package logs

import (
	"sync"
	"syscall"

	"golang.org/x/text/encoding/charmap"
)

var oemCharmap = sync.OnceValue(func() *charmap.Charmap {
	r, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetOEMCP").Call()
	return charmapFor(uint32(r))
})

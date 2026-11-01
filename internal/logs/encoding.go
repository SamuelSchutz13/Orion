package logs

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// Clean garante UTF-8 válido para o JSON nunca quebrar. Texto que não é UTF-8 (ex.: saída do cmd
// em cp850) é decodificado pela code page OEM do sistema; sem ela, bytes inválidos viram "�".
func Clean(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	if cm := oemCharmap(); cm != nil {
		return decodeWith(cm, s)
	}
	return strings.ToValidUTF8(s, "�")
}

func decodeWith(cm *charmap.Charmap, s string) string {
	out, err := cm.NewDecoder().String(s)
	if err != nil {
		return strings.ToValidUTF8(s, "�")
	}
	return out
}

func charmapFor(cp uint32) *charmap.Charmap {
	switch cp {
	case 437:
		return charmap.CodePage437
	case 850:
		return charmap.CodePage850
	case 852:
		return charmap.CodePage852
	case 866:
		return charmap.CodePage866
	case 1252:
		return charmap.Windows1252
	}
	return nil
}

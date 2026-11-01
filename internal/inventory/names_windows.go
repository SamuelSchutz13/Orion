package inventory

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// snapshotNames lê os nomes de todos os processos de uma vez pelo Toolhelp,
// que funciona mesmo para processos protegidos onde abrir o handle falha.
func snapshotNames() map[int32]string {
	h, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(h)
	names := map[int32]string{}
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err = windows.Process32First(h, &e); err == nil; err = windows.Process32Next(h, &e) {
		names[int32(e.ProcessID)] = windows.UTF16ToString(e.ExeFile[:])
	}
	return names
}

package inventory

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

type snapEntry struct {
	name string
	ppid int32
}

// snapshotProcs lê nome e PID do pai de todos os processos de uma vez pelo Toolhelp,
// que funciona mesmo para processos protegidos onde abrir o handle falha.
func snapshotProcs() map[int32]snapEntry {
	h, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(h)
	procs := map[int32]snapEntry{}
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err = windows.Process32First(h, &e); err == nil; err = windows.Process32Next(h, &e) {
		procs[int32(e.ProcessID)] = snapEntry{windows.UTF16ToString(e.ExeFile[:]), int32(e.ParentProcessID)}
	}
	return procs
}

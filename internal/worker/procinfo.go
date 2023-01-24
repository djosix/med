package worker

type ProcKind string

const (
	ProcKind_None     ProcKind = "None"
	ProcKind_Example  ProcKind = "Example"
	ProcKind_Main     ProcKind = "Main"
	ProcKind_Exec     ProcKind = "Exec"
	ProcKind_Term     ProcKind = "Term"
	ProcKind_LocalPF  ProcKind = "LocalPF"
	ProcKind_RemotePF ProcKind = "RemotePF"
	ProcKind_Upload   ProcKind = "Upload"
	ProcKind_Download ProcKind = "Download"
)

type ProcSide byte

const (
	ProcSide_None   ProcSide = 0b00
	ProcSide_Client ProcSide = 0b01
	ProcSide_Server ProcSide = 0b10
	ProcSide_Both   ProcSide = 0b11
)

type ProcInfo struct {
	kind ProcKind
	side ProcSide
}

func NewProcInfo(kind ProcKind, side ProcSide) ProcInfo {
	return ProcInfo{kind: kind, side: side}
}

func (p ProcInfo) Kind() ProcKind {
	return p.kind
}

func (p ProcInfo) Side() ProcSide {
	return p.side
}

package worker

type ProcKind string

const (
	ProcKind_None     ProcKind = "none"
	ProcKind_Example  ProcKind = "example"
	ProcKind_Main     ProcKind = "main"
	ProcKind_Exec     ProcKind = "exec"
	ProcKind_Get      ProcKind = "get"
	ProcKind_Put      ProcKind = "put"
	ProcKind_FTP      ProcKind = "ftp" // TODO
	ProcKind_LocalPF  ProcKind = "forward-local"
	ProcKind_RemotePF ProcKind = "forward-remote"
	ProcKind_Socks    ProcKind = "socks"
	ProcKind_Proxy    ProcKind = "proxy" // TODO
	ProcKind_Self     ProcKind = "self"
	ProcKind_WebUI    ProcKind = "webui" // TODO
	ProcKind_CLI      ProcKind = "cli"   // TODO
	ProcKind_TMUX     ProcKind = "tmux"  // TODO
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

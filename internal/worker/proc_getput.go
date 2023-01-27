package worker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/djosix/med/internal/logger"
)

type GetPutSpec struct {
	SourcePaths []string
	DestPath    string
	IsTarMode   bool
	IsGzipMode  bool
}

// Client

type GetProcClient struct {
	ProcInfo
	spec GetPutSpec
}

func NewGetProcClient(spec GetPutSpec) *GetProcClient {
	return &GetProcClient{
		ProcInfo: NewProcInfo(ProcKind_Get, ProcSide_Client),
		spec:     spec,
	}
}

func (p *GetProcClient) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger("GetProcClient")
	logger.Debug("start")
	defer logger.Debug("done")

	if err := initGetPutDestPath(p.spec); err != nil {
		logger.Error(err)
		return
	}

	SendProcSpec(ctx, p.spec)
}

// Server

type GetProcServer struct {
	ProcInfo
}

func NewGetProcServer() *GetProcServer {
	return &GetProcServer{
		ProcInfo: NewProcInfo(ProcKind_Get, ProcSide_Server),
	}
}

func (p *GetProcServer) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger("GetProcServer")

	// Get spec
	spec, err := RecvProcSpec[GetPutSpec](ctx)
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debug("spec =", spec)

	// ctx.PktOutCh <- helper.NewDataPacket([]byte(fmt.Sprintf("Hello %s from server", spec.Name)))
	logger.Info(string((<-ctx.PktInCh).Data))
}

//

func initGetPutDestPath(spec GetPutSpec) error {
	stat, err := os.Stat(spec.DestPath)

	var isDestDir bool
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(spec.DestPath, 0755); err != nil {
				return err
			}
			isDestDir = true
		} else {
			return err
		}
	} else {
		isDestDir = stat.IsDir()
	}

	if !spec.IsTarMode {
		// normal mode like cp

		if len(spec.SourcePaths) > 1 {
			if !isDestDir {
				return fmt.Errorf("cannot copy multiple entries to a file, try tar mode")
			}

			fnames := map[string]struct{}{}
			for _, path := range spec.SourcePaths {
				base := filepath.Base(path)
				if _, exists := fnames[base]; exists {
					return fmt.Errorf("cannot copy files with same names to a dir, try tar mode")
				}
				fnames[base] = struct{}{}
			}
		}

	} else {
		// tar mode

		if !isDestDir {
			return fmt.Errorf("cannot extract tar to a file")
		}
	}

	return nil
}

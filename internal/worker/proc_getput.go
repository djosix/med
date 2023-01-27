package worker

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/logger"
	"github.com/muesli/cancelreader"
)

type GetPutSpec struct {
	SourcePaths []string
	DestPath    string
	IsTarMode   bool
	IsGzipMode  bool
}

// Get Client

type GetProcClient struct {
	ProcInfo
	spec GetPutSpec
}

func NewGetProcClient(spec GetPutSpec) *GetProcClient {
	preprocessGetPutSpec(&spec)
	return &GetProcClient{
		ProcInfo: NewProcInfo(ProcKind_Get, ProcSide_Client),
		spec:     spec,
	}
}

func (p *GetProcClient) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger("GetProcClient")
	logger.Debug("start")
	defer logger.Debug("done")

	destPaths, err := initGetPutRecvSide(&p.spec)
	if err != nil {
		logger.Error(err)
		return
	}

	// Remove client info in spec before send
	spec := p.spec
	spec.DestPath = ""
	SendProcSpec(ctx, p.spec)

	for err := range recvFiles(ctx, destPaths) {
		logger.Error("error:", err)
	}
}

// Get Server

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
	logger.Debug("start")
	defer logger.Debug("done")

	spec, err := RecvProcSpec[GetPutSpec](ctx)
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debugf("spec: %#v", spec)

	for i, path := range spec.SourcePaths {
		err := sendFile(ctx, i, path)
		if err == ErrLoopClosed {
			return
		} else if err != nil {
			logger.Debugf("send file [%v]: %v", path, err)
		}
		if !sendFileEnd(ctx, i, path, err) {
			return
		}
	}
}

// Put Client

type PutProcClient struct {
	ProcInfo
	spec GetPutSpec
}

func NewPutProcClient(spec GetPutSpec) *PutProcClient {
	return &PutProcClient{
		ProcInfo: NewProcInfo(ProcKind_Get, ProcSide_Client),
		spec:     spec,
	}
}

func (p *PutProcClient) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger("PutProcClient")
	logger.Debug("start")
	defer logger.Debug("done")

	SendProcSpec(ctx, p.spec)

	for i, path := range p.spec.SourcePaths {
		err := sendFile(ctx, i, path)
		if err == ErrLoopClosed {
			return
		} else if err != nil {
			logger.Debugf("send file [%v]: %v", i, path, err)
		}
		if !sendFileEnd(ctx, i, path, err) {
			return
		}
	}
}

// Put Server

type PutProcServer struct {
	ProcInfo
}

func NewPutProcServer() *PutProcServer {
	return &PutProcServer{
		ProcInfo: NewProcInfo(ProcKind_Get, ProcSide_Server),
	}
}

func (p *PutProcServer) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger("PutProcServer")
	logger.Debug("start")
	defer logger.Debug("done")

	// Get spec
	spec, err := RecvProcSpec[GetPutSpec](ctx)
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debugf("spec: %#v", spec)

	destPaths, err := initGetPutRecvSide(spec)
	if err != nil {
		logger.Error(err)
		return
	}

	for err := range recvFiles(ctx, destPaths) {
		logger.Error("error:", err)
	}
}

//

func preprocessGetPutSpec(spec *GetPutSpec) {
	if spec.DestPath == "" {
		spec.DestPath = "."
	}

	// Remove duplicate source paths
	{
		pathSet := helper.NewSet[string]()
		pathSet.Add(spec.SourcePaths...)
		spec.SourcePaths = pathSet.Values()
	}
}

func initGetPutRecvSide(spec *GetPutSpec) (destPaths []string, err error) {
	if len(spec.SourcePaths) == 0 {
		return nil, fmt.Errorf("no source path")
	}

	stat, err := os.Stat(spec.DestPath)

	var isDestDir bool
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(spec.DestPath, 0755); err != nil {
				return nil, err
			}
			isDestDir = true
		} else {
			return nil, err
		}
	} else {
		isDestDir = stat.IsDir()
	}

	if !spec.IsTarMode {
		// normal mode like cp

		if len(spec.SourcePaths) > 1 {
			if !isDestDir {
				return nil, fmt.Errorf("cannot copy multiple entries to a file, try tar mode")
			}

			baseSet := helper.NewSet[string]()
			for _, path := range spec.SourcePaths {
				base := filepath.Base(path)
				if baseSet.Has(base) {
					return nil, fmt.Errorf("cannot copy files with same names to a dir, try tar mode")
				}
				baseSet.Add(base)
				destPaths = append(destPaths, filepath.Join(spec.DestPath, base))
			}

			return destPaths, nil

		} else {
			destPath := spec.DestPath
			if isDestDir {
				destPath = filepath.Join(destPath, filepath.Base(spec.SourcePaths[0]))
			}
			return append(destPaths, destPath), nil
		}

	} else {
		// tar mode

		if !isDestDir {
			return nil, fmt.Errorf("cannot extract tar to a file")
		}

		return append(destPaths, spec.DestPath), nil
	}
}

func sendFile(ctx *ProcRunCtx, idx int, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r, err := cancelreader.NewReader(f)
	if err != nil {
		return err
	}
	defer r.Close()

	{
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		go func() { <-ctx.Done(); r.Cancel() }()
	}

	data := make([]byte, 4096)
	var headerLen int
	{
		data[0] = 0 // 0=data
		header := binary.AppendUvarint([]byte{0}, uint64(idx))
		copy(data[1:], header)
		headerLen = len(header)
	}
	// bytes.NewReader()

	for {
		n, err := r.Read(data[headerLen:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		if !ctx.Output(helper.NewDataPacket(data[:headerLen+n])) {
			return ErrLoopClosed
		}
	}
}

func sendFileEnd(ctx *ProcRunCtx, idx int, path string, err error) bool {
	data := []byte{1} // 1=end
	data = binary.AppendUvarint(data, uint64(idx))
	if err != nil {
		data = append(data, []byte(err.Error())...)
	}
	return ctx.Output(helper.NewDataPacket(data))
}

func recvFiles(ctx *ProcRunCtx, destPaths []string) <-chan error {
	ch := make(chan error)

	go func() {
		defer close(ch)

		files := make([]*os.File, len(destPaths))
		done := make([]bool, len(destPaths))
		doneCount := 0

		for doneCount < len(destPaths) {
			pkt, ok := <-ctx.PktInCh
			if !ok {
				ch <- io.EOF
				return
			}

			r := bytes.NewReader(pkt.Data)

			var end bool
			{
				b, err := r.ReadByte()
				if err != nil {
					ch <- err
					return
				}
				end = (b == 1)
			}

			var idx int
			{
				b, err := binary.ReadUvarint(r)
				if err != nil {
					ch <- fmt.Errorf("cannot get file index")
					return
				}
				if b > uint64(len(destPaths)) {
					ch <- fmt.Errorf("invalid file index")
					return
				}
				idx = int(b)
			}

			if !end {
				if done[idx] == true {
					logger.Warn("receive data for a done file")
					continue
				}

				if files[idx] == nil {
					newf, err := os.OpenFile(destPaths[idx], os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
					if err != nil {
						ch <- err
						return
					}
					defer newf.Close()

					files[idx] = newf
				}

				r.WriteTo(files[idx])
			} else {
				if files[idx] != nil {
					files[idx].Close()
					files[idx] = nil
				}

				if !done[idx] {
					done[idx] = true
					doneCount++
				} else {
					logger.Warn("receive end message twice")
				}

				if r.Len() > 0 {
					buf := bytes.NewBuffer(nil)
					r.WriteTo(buf)
					errMsg := string(buf.Bytes())

					ch <- fmt.Errorf("receive [%v] remote: %v", destPaths[idx], errMsg)
				}
			}
		}
	}()

	return ch
}

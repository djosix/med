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
	pb "github.com/djosix/med/internal/protobuf"
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
	initGetPutSpec(&spec)
	return &GetProcClient{
		ProcInfo: NewProcInfo(ProcKind_Get, ProcSide_Client),
		spec:     spec,
	}
}

func (p *GetProcClient) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger(string(p.Kind()))
	logger.Debug("start")
	defer logger.Debug("done")

	dstPaths, err := initGetPutRecvSide(&p.spec)
	if err != nil {
		logger.Error(err)
		return
	}

	// Remove client info in spec before send
	spec := p.spec
	spec.DestPath = ""
	SendProcSpec(ctx, p.spec)

	for result := range recvFiles(ctx, spec.SourcePaths, dstPaths) {
		if result.dst == "" {
			logger.Errorf("error: %v", result.err)
		} else if result.err != nil {
			logger.Errorf("remote %#v to %#v error: %v", result.src, result.dst, result.err)
		} else {
			logger.Infof("remote %#v to %#v (success)", result.src, result.dst)
		}
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
	logger := logger.NewLogger(string(p.Kind()))
	logger.Debug("start")
	defer logger.Debug("done")

	spec, err := RecvProcSpec[GetPutSpec](ctx)
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debugf("spec: %#v", spec)

	for i, path := range spec.SourcePaths {
		err := sendFileData(ctx, i, path)
		logger.Debugf("sent data [%v] i=%v err=%v", path, i, err)
		if err == ErrLoopClosed {
			logger.Debug("ErrLoopClosed")
			return
		} else if err != nil {
			logger.Debugf("send file [%v]: %v", path, err)
		}
		if !sendFileEnd(ctx, i, path, err) {
			return
		}
		logger.Debugf("sent end [%v] i=%v err=%v", path, i, err)
	}
}

// Put Client

type PutProcClient struct {
	ProcInfo
	spec GetPutSpec
}

func NewPutProcClient(spec GetPutSpec) *PutProcClient {
	initGetPutSpec(&spec)
	return &PutProcClient{
		ProcInfo: NewProcInfo(ProcKind_Get, ProcSide_Client),
		spec:     spec,
	}
}

func (p *PutProcClient) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger(string(p.Kind()))
	logger.Debug("start")
	defer logger.Debug("done")

	SendProcSpec(ctx, p.spec)

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			pkt := ctx.InputPacket()
			if pkt == nil {
				return
			}
			buf := bytes.NewBuffer(pkt.Data)
			var srcPath string
			{
				u, err := binary.ReadUvarint(buf)
				if err != nil {
					logger.Error("cannot read file index")
					continue
				}
				srcPath = p.spec.SourcePaths[int(u)]
			}
			var dstPath string
			var errStr string
			{
				u, err := binary.ReadUvarint(buf)
				if err != nil {
					logger.Error("cannot read file index")
					continue
				}
				buf := buf.Bytes()
				dstPath = string(buf[:int(u)])
				errStr = string(buf[int(u):])
			}
			if errStr == "" {
				logger.Infof("copy %#v to remote %#v (success)", srcPath, dstPath)
			} else {
				logger.Infof("copy %#v to remote %#v error: %v", srcPath, dstPath, errStr)
			}
		}
	}()

	for i, path := range p.spec.SourcePaths {
		err := sendFileData(ctx, i, path)
		if err == ErrLoopClosed {
			return
		} else if err != nil {
			logger.Debugf("send file [%v]: %v", i, path, err)
		}
		if !sendFileEnd(ctx, i, path, err) {
			return
		}
	}

	<-done
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
	logger := logger.NewLogger(string(p.Kind()))
	logger.Debug("start")
	defer logger.Debug("done")

	// Get spec
	spec, err := RecvProcSpec[GetPutSpec](ctx)
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debugf("spec: %#v", spec)

	dstPaths, err := initGetPutRecvSide(spec)
	if err != nil {
		logger.Error(err)
		return
	}

	makeRespPkt := func(idx int, dstPath string, err error) *pb.Packet {
		buf := []byte{}
		buf = binary.AppendUvarint(buf, uint64(idx))
		buf = binary.AppendUvarint(buf, uint64(len(dstPath)))
		buf = append(buf, []byte(dstPath)...)
		if err != nil {
			buf = append(buf, []byte(err.Error())...)
		}
		return helper.NewDataPacket(buf)
	}

	for result := range recvFiles(ctx, spec.SourcePaths, dstPaths) {
		if result.dst == "" && result.err != nil {
			ctx.PacketOutput(helper.NewErrorPacket(err.Error()))
		} else {
			ctx.PacketOutput(makeRespPkt(result.idx, result.dst, result.err))
		}
	}
}

//

func initGetPutSpec(spec *GetPutSpec) {
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

	var exists bool = false
	var isDir bool
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	} else {
		exists = true
		isDir = stat.IsDir()
	}

	if !spec.IsTarMode {
		// normal mode like cp

		if len(spec.SourcePaths) > 1 {
			if !exists {
				if err := os.MkdirAll(spec.DestPath, 0755); err != nil {
					return nil, err
				}
				isDir = true
			}
			if !isDir {
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
			if isDir {
				destPath = filepath.Join(destPath, filepath.Base(spec.SourcePaths[0]))
			}
			return append(destPaths, destPath), nil
		}

	} else {
		// tar mode

		if !isDir {
			return nil, fmt.Errorf("cannot extract tar to a file")
		}

		return append(destPaths, spec.DestPath), nil
	}
}

func sendFileData(ctx *ProcRunCtx, idx int, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	recvCtx, recvCancel := context.WithCancel(ctx)
	defer recvCancel()
	go func() { <-recvCtx.Done(); f.Close() }()

	data := make([]byte, 4096)
	var headerLen int
	{
		header := binary.AppendUvarint([]byte{0}, uint64(idx))
		copy(data, header)
		headerLen = len(header)
	}
	// bytes.NewReader()

	for {
		n, err := f.Read(data[headerLen:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		if !ctx.PacketOutput(helper.NewDataPacket(data[:headerLen+n])) {
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
	return ctx.PacketOutput(helper.NewDataPacket(data))
}

type recvFileResult struct {
	idx int
	src string
	dst string
	err error
}

func recvFiles(ctx *ProcRunCtx, srcPaths []string, dstPaths []string) <-chan *recvFileResult {
	ch := make(chan *recvFileResult)

	go func() {
		defer close(ch)

		files := make([]*os.File, len(dstPaths))
		done := make([]bool, len(dstPaths))
		doneCount := 0

		for doneCount < len(dstPaths) {
			pkt := ctx.InputPacket()
			if pkt == nil {
				err := fmt.Errorf("cannot read packet")
				ch <- &recvFileResult{err: err}
				return
			}

			r := bytes.NewReader(pkt.Data)

			var end bool
			{
				b, err := r.ReadByte()
				if err != nil {
					err = fmt.Errorf("cannot read byte: %w", err)
					ch <- &recvFileResult{err: err}
					return
				}
				end = (b == 1)
			}

			var idx int
			{
				b, err := binary.ReadUvarint(r)
				if err != nil {
					err = fmt.Errorf("cannot get file index: %w", err)
					ch <- &recvFileResult{err: err}
					return
				}
				if b > uint64(len(dstPaths)) {
					err = fmt.Errorf("invalid file index: %v/%v", b, len(dstPaths))
					ch <- &recvFileResult{err: err}
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
					newf, err := os.OpenFile(dstPaths[idx], os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
					if err != nil {
						err = fmt.Errorf("open [%v]:", dstPaths[idx])
						ch <- &recvFileResult{err: err}
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

				result := &recvFileResult{
					idx: idx,
					src: srcPaths[idx],
					dst: dstPaths[idx],
				}

				if r.Len() > 0 {
					buf := bytes.NewBuffer(nil)
					r.WriteTo(buf)
					errMsg := string(buf.Bytes())

					result.err = fmt.Errorf("remote: %v", errMsg)
				}

				ch <- result
			}
		}
	}()

	return ch
}

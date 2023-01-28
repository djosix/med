package worker

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/logger"
	"github.com/djosix/med/internal/protobuf"
)

type LocalPFSpec struct {
	LocalEndpoint  string
	RemoteEndpoint string
}

// Client

type LocalPFProcClient struct {
	ProcInfo
	spec LocalPFSpec
}

func NewLocalPFProcClient(spec LocalPFSpec) *LocalPFProcClient {
	return &LocalPFProcClient{
		ProcInfo: NewProcInfo(ProcKind_LocalPF, ProcSide_Client),
		spec:     spec,
	}
}

func (p *LocalPFProcClient) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger(string(p.Kind()))

	// Send spec
	SendProcSpec(ctx, p.spec)

	listener, err := net.Listen(helper.SplitEndpoint(p.spec.LocalEndpoint))
	if err != nil {
		logger.Error("listen:", err)
		return
	}
	defer listener.Close()

	reader := func(idx int, conn net.Conn, bufOutCh chan<- []byte) {
		logger := logger.NewLogger(fmt.Sprintf("reader[%v]", idx))
		logger.Debug("start")
		defer logger.Debug("done")

		data := make([]byte, 4096)
		hLen := copy(data, binary.AppendUvarint([]byte{}, uint64(idx)))
		for {
			logger.Debugf("read")
			n, err := conn.Read(data[hLen:])
			if err != nil {
				logger.Error("read:", err)
				break
			}
			logger.Debugf("read: %v", data[:hLen+n])
			bufOutCh <- helper.Clone(data[:hLen+n])
		}
		bufOutCh <- data[:hLen]
	}

	writer := func(idx int, conn net.Conn, bufInCh <-chan []byte) {
		logger := logger.NewLogger(fmt.Sprintf("writer[%v]", idx))
		logger.Debug("start")
		defer logger.Debug("done")

		for buf := range bufInCh {
			logger.Debugf("write to conn: %#v", buf)
			n, err := conn.Write(buf)
			if n != len(buf) || err != nil {
				logger.Error("write:", err, n)
				break
			}
		}
	}

	ctx1, _ := context.WithCancel(ctx)
	wg := sync.WaitGroup{}

	bufOutCh := make(chan []byte)
	bufInChMap := map[int]chan []byte{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; true; i++ {
			idx := i

			conn, err := listener.Accept()
			if err != nil {
				logger.Error("accept:", err)
				return
			}
			logger.Debug("accept:", conn.RemoteAddr())

			wg.Add(1)
			go func() {
				defer wg.Done()
				reader(idx, conn, bufOutCh)
			}()

			bufInCh := make(chan []byte)
			bufInChMap[idx] = bufInCh

			wg.Add(1)
			go func() {
				defer wg.Done()
				writer(idx, conn, bufInCh)
			}()

			go func() {
				<-ctx1.Done()
				logger.Debug("close conn", idx)
				conn.Close()
			}()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			pkt := ctx.InputPacket()
			if pkt == nil {
				return
			}

			if pkt.Kind == protobuf.PacketKind_PacketKindData {
				buf := bytes.NewBuffer(pkt.Data)
				u, err := binary.ReadUvarint(buf)
				if err != nil {
					logger.Error("cannot read idx")
					continue
				}
				idx := int(u)
				if bufInCh, ok := bufInChMap[idx]; ok {
					data := buf.Bytes()
					if len(data) > 0 {
						logger.Debugf("send to conn %v: %#v", idx, data)
						bufInCh <- data
					} else {
						logger.Debugf("close to conn %v", idx)
						close(bufInCh)
						delete(bufInChMap, idx)
					}
				} else {
					logger.Debugf("conn %v not exist", idx)
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger := logger.NewLogger("output")
		logger.Debug("start")
		defer logger.Debug("done")

		for buf := range bufOutCh {
			if !ctx.OutputPacket(helper.NewDataPacket(buf)) {
				break
			}
		}
	}()

	wg.Wait()
}

// Server

type LocalPFProcServer struct {
	ProcInfo
}

func NewLocalPFProcServer() *LocalPFProcServer {
	return &LocalPFProcServer{
		ProcInfo: NewProcInfo(ProcKind_LocalPF, ProcSide_Server),
	}
}

func (p *LocalPFProcServer) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger(string(p.Kind()))

	// Get spec
	spec, err := RecvProcSpec[LocalPFSpec](ctx)
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debug("spec:", spec)
	_ = spec

	connect := func() (net.Conn, error) {
		return net.Dial(helper.SplitEndpoint(spec.RemoteEndpoint))
	}

	conns := map[int]chan<- []byte{}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			pkt := ctx.InputPacket()
			if pkt == nil {
				return
			}

			if pkt.Kind != protobuf.PacketKind_PacketKindData {
				continue
			}

			dataBuf := bytes.NewBuffer(pkt.Data)
			u, err := binary.ReadUvarint(dataBuf)
			if err != nil {
				logger.Error("cannot read idx:", err)
				continue
			}

			idx := int(u)
			data := dataBuf.Bytes()

			logger.Debugf("recv idx=%v n=%v s=%s %#v", idx, len(data), string(data), data)

			if bufInCh, ok := conns[idx]; ok {
				if len(data) == 0 {
					logger.Debugf("close conn %v", idx)
					close(bufInCh)
					delete(conns, idx)
				} else {
					logger.Debugf("send to conn %v", idx)
					bufInCh <- data
				}
			} else {
				if len(data) > 0 {
					logger.Debugf("create conn %v", idx)

					bufInCh := make(chan []byte)
					conns[idx] = bufInCh

					wg.Add(1)
					go func() {
						defer wg.Done()

						conn, err := connect()
						if err != nil {
							logger.Error("connect:", err)
							return
						}
						defer conn.Close()

						logger.Debugf("created conn %v", idx)

						wg := sync.WaitGroup{}

						wg.Add(1)
						go func() {
							defer wg.Done()

							logger := logger.NewLogger(fmt.Sprintf("writer[%v]", idx))
							logger.Debug("start")
							defer logger.Debug("done")

							for buf := range bufInCh {
								logger.Errorf("conn %v write: %#v", idx, buf)
								if n, err := conn.Write(buf); n != len(buf) || err != nil {
									logger.Errorf("conn %v write: %v %v", idx, n, err)
									return
								}
								logger.Errorf("conn %v written", idx)
							}
						}()

						wg.Add(1)
						go func() {
							defer wg.Done()

							logger := logger.NewLogger(fmt.Sprintf("reader[%v]", idx))
							logger.Debug("start")
							defer logger.Debug("done")

							data := make([]byte, 4096)
							hLen := copy(data, binary.AppendUvarint([]byte{}, uint64(idx)))

							for {
								n, err := conn.Read(data[hLen:])
								if n == 0 || err != nil {
									logger.Error("conn read:", n, err)
									return
								}

								logger.Debugf("from conn %v: %#v", idx, data[:hLen+n])

								if !ctx.OutputPacket(helper.NewDataPacket(data[:hLen+n])) {
									break
								}
							}
						}()

						logger.Debugf("send to conn %v", idx)
						bufInCh <- data

						wg.Wait()
					}()
				}
			}
		}
	}()

	wg.Wait()
}

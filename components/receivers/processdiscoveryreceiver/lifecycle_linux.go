//go:build linux

package processdiscoveryreceiver

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"go.uber.org/zap"
)

const lifecycleRecordSize = 32

type linuxLifecycleSource struct {
	bufferSize     int
	logger         *zap.Logger
	reader         *ringbuf.Reader
	links          []link.Link
	maps           []*ebpf.Map
	programs       []*ebpf.Program
	once           sync.Once
	networkEnabled bool
}

type bpfLifecycleRecord struct {
	Type      uint32
	PID       uint32
	ParentPID uint32
	FD        int32
	Timestamp uint64
	CgroupID  uint64
}

func newLifecycleSource(cfg *Config, logger *zap.Logger) (LifecycleSource, error) {
	if !cfg.Lifecycle.Enabled {
		return disabledLifecycleSource{}, nil
	}
	return &linuxLifecycleSource{bufferSize: cfg.Lifecycle.BufferSize, logger: logger, networkEnabled: cfg.Network.Enabled}, nil
}

func (s *linuxLifecycleSource) Start(ctx context.Context) (<-chan LifecycleEvent, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		s.logger.Debug("unable to raise memlock limit", zap.Error(err))
	}
	mapSize := nextPowerOfTwo(max(s.bufferSize*lifecycleRecordSize, os.Getpagesize()))
	eventsMap, err := ebpf.NewMap(&ebpf.MapSpec{Name: "process_lifecycle", Type: ebpf.RingBuf, MaxEntries: uint32(mapSize)})
	if err != nil {
		return nil, fmt.Errorf("create lifecycle ring buffer: %w", err)
	}
	s.maps = append(s.maps, eventsMap)
	reader, err := ringbuf.NewReader(eventsMap)
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("create lifecycle reader: %w", err)
	}
	s.reader = reader

	execProgram, err := ebpf.NewProgram(lifecycleProgramSpec("process_exec", eventsMap, 1))
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("load sched_process_exec program: %w", err)
	}
	exitProgram, err := ebpf.NewProgram(lifecycleProgramSpec("process_exit", eventsMap, 2))
	if err != nil {
		execProgram.Close()
		s.Close()
		return nil, fmt.Errorf("load sched_process_exit program: %w", err)
	}
	s.programs = append(s.programs, execProgram, exitProgram)
	execLink, err := link.Tracepoint("sched", "sched_process_exec", execProgram, nil)
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("attach sched_process_exec: %w", err)
	}
	exitLink, err := link.Tracepoint("sched", "sched_process_exit", exitProgram, nil)
	if err != nil {
		execLink.Close()
		s.Close()
		return nil, fmt.Errorf("attach sched_process_exit: %w", err)
	}
	s.links = append(s.links, execLink, exitLink)
	if s.networkEnabled {
		for _, probe := range []struct {
			name      string
			eventType int32
			entry     bool
		}{{"sys_enter_accept", 4, true}, {"sys_enter_accept4", 4, true}, {"sys_exit_accept", 3, false}, {"sys_exit_accept4", 3, false}} {
			program, loadErr := ebpf.NewProgram(acceptProgramSpec(eventsMap, probe.eventType, probe.entry))
			if loadErr != nil {
				s.logger.Debug("unable to load accept tracepoint", zap.String("event", probe.name), zap.Error(loadErr))
				continue
			}
			s.programs = append(s.programs, program)
			acceptLink, attachErr := link.Tracepoint("syscalls", probe.name, program, nil)
			if attachErr != nil {
				s.logger.Debug("unable to attach accept tracepoint", zap.String("event", probe.name), zap.Error(attachErr))
				continue
			}
			s.links = append(s.links, acceptLink)
		}
	}

	output := make(chan LifecycleEvent, s.bufferSize)
	go s.read(ctx, output)
	return output, nil
}

func acceptProgramSpec(eventMap *ebpf.Map, eventType int32, entry bool) *ebpf.ProgramSpec {
	valueOffset := int16(16)
	if entry {
		valueOffset = 16
	}
	ins := asm.Instructions{
		asm.LoadMem(asm.R8, asm.R1, valueOffset, asm.DWord),
	}
	if !entry {
		ins = append(ins, asm.JSLT.Imm(asm.R8, 0, "skip_output"))
	}
	ins = append(ins,
		asm.Mov.Imm(asm.R1, eventType),
		asm.StoreMem(asm.RFP, -32, asm.R1, asm.Word),
		asm.FnGetCurrentPidTgid.Call(),
		asm.Mov.Reg(asm.R6, asm.R0),
		asm.RSh.Imm(asm.R6, 32),
		asm.StoreMem(asm.RFP, -28, asm.R6, asm.Word),
		asm.StoreMem(asm.RFP, -24, asm.R7, asm.Word),
		asm.StoreMem(asm.RFP, -20, asm.R8, asm.Word),
		asm.FnKtimeGetNs.Call(),
		asm.StoreMem(asm.RFP, -16, asm.R0, asm.DWord),
		asm.FnGetCurrentCgroupId.Call(),
		asm.StoreMem(asm.RFP, -8, asm.R0, asm.DWord),
		asm.LoadMapPtr(asm.R1, eventMap.FD()),
		asm.Mov.Reg(asm.R2, asm.RFP),
		asm.Add.Imm(asm.R2, -32),
		asm.Mov.Imm(asm.R3, lifecycleRecordSize),
		asm.Mov.Imm(asm.R4, 0),
		asm.FnRingbufOutput.Call(),
		asm.Mov.Imm(asm.R0, 0),
		asm.Return().WithSymbol("skip_output"),
	)
	return &ebpf.ProgramSpec{Name: "process_accept", Type: ebpf.TracePoint, License: "GPL", Instructions: ins}
}

func lifecycleProgramSpec(name string, eventMap *ebpf.Map, eventType int32) *ebpf.ProgramSpec {
	ins := asm.Instructions{
		asm.Mov.Imm(asm.R1, eventType),
		asm.StoreMem(asm.RFP, -32, asm.R1, asm.Word),
		asm.FnGetCurrentPidTgid.Call(),
		asm.Mov.Reg(asm.R6, asm.R0),
		asm.Mov.Reg(asm.R7, asm.R0),
		asm.RSh.Imm(asm.R6, 32),
		asm.StoreMem(asm.RFP, -28, asm.R6, asm.Word),
		asm.Mov.Imm(asm.R1, 0),
		asm.StoreMem(asm.RFP, -24, asm.R1, asm.Word),
		asm.StoreMem(asm.RFP, -20, asm.R1, asm.Word),
		asm.FnKtimeGetNs.Call(),
		asm.StoreMem(asm.RFP, -16, asm.R0, asm.DWord),
		asm.FnGetCurrentCgroupId.Call(),
		asm.StoreMem(asm.RFP, -8, asm.R0, asm.DWord),
	}
	if eventType == 2 {
		ins = append(ins,
			asm.JNE.Reg32(asm.R6, asm.R7, "skip_output"),
		)
	}
	ins = append(ins,
		asm.LoadMapPtr(asm.R1, eventMap.FD()),
		asm.Mov.Reg(asm.R2, asm.RFP),
		asm.Add.Imm(asm.R2, -32),
		asm.Mov.Imm(asm.R3, lifecycleRecordSize),
		asm.Mov.Imm(asm.R4, 0),
		asm.FnRingbufOutput.Call(),
		asm.Mov.Imm(asm.R0, 0),
		asm.Return().WithSymbol("skip_output"),
	)
	return &ebpf.ProgramSpec{Name: name, Type: ebpf.TracePoint, License: "GPL", Instructions: ins}
}

func (s *linuxLifecycleSource) read(ctx context.Context, output chan<- LifecycleEvent) {
	defer close(output)
	for {
		record, err := s.reader.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) || ctx.Err() != nil {
				return
			}
			sendLifecycle(ctx, output, LifecycleEvent{Type: lifecycleLoss, Lost: 1, ObservedAt: time.Now()})
			continue
		}
		var raw bpfLifecycleRecord
		if len(record.RawSample) < lifecycleRecordSize || binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &raw) != nil {
			sendLifecycle(ctx, output, LifecycleEvent{Type: lifecycleLoss, Lost: 1, ObservedAt: time.Now()})
			continue
		}
		eventType := lifecycleExec
		if raw.Type == 2 {
			eventType = lifecycleExit
		} else if raw.Type == 3 {
			eventType = lifecycleAccept
		} else if raw.Type == 4 {
			eventType = lifecycleAcceptEnter
		}
		sendLifecycle(ctx, output, LifecycleEvent{Type: eventType, PID: int32(raw.PID), ThreadID: int32(raw.ParentPID), FD: raw.FD, ObservedAt: time.Now(), CgroupID: raw.CgroupID})
	}
}

func sendLifecycle(ctx context.Context, output chan<- LifecycleEvent, event LifecycleEvent) {
	select {
	case output <- event:
	default:
		select {
		case output <- LifecycleEvent{Type: lifecycleLoss, Lost: 1, ObservedAt: time.Now()}:
		default:
		}
	case <-ctx.Done():
	}
}

func (s *linuxLifecycleSource) Close() error {
	var closeErr error
	s.once.Do(func() {
		if s.reader != nil {
			closeErr = errors.Join(closeErr, s.reader.Close())
		}
		for _, item := range s.links {
			closeErr = errors.Join(closeErr, item.Close())
		}
		for _, item := range s.programs {
			closeErr = errors.Join(closeErr, item.Close())
		}
		for _, item := range s.maps {
			closeErr = errors.Join(closeErr, item.Close())
		}
	})
	return closeErr
}

func nextPowerOfTwo(value int) int {
	value--
	for shift := 1; shift < 64; shift <<= 1 {
		value |= value >> shift
	}
	return value + 1
}

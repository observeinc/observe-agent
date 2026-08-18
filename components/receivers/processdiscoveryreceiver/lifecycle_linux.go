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
	"syscall"
	"time"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"go.uber.org/zap"
)

const lifecycleRecordSize = 32

type linuxLifecycleSource struct {
	bufferSize int
	logger     *zap.Logger
	reader     *ringbuf.Reader
	objs       *lifecycleObjects
	links      []link.Link
	once       sync.Once
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
	return &linuxLifecycleSource{bufferSize: cfg.Lifecycle.BufferSize, logger: logger}, nil
}

func (s *linuxLifecycleSource) Start(ctx context.Context) (<-chan LifecycleEvent, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		s.logger.Debug("unable to raise memlock limit", zap.Error(err))
	}
	ensureTracefs(s.logger)

	spec, err := loadLifecycle()
	if err != nil {
		return nil, fmt.Errorf("load lifecycle BPF spec: %w", err)
	}

	mapSize := nextPowerOfTwo(max(s.bufferSize*lifecycleRecordSize, os.Getpagesize()))
	spec.Maps["process_lifecycle"].MaxEntries = uint32(mapSize)

	var objs lifecycleObjects
	if err := spec.LoadAndAssign(&objs, nil); err != nil {
		return nil, fmt.Errorf("load lifecycle BPF objects: %w", err)
	}
	s.objs = &objs

	reader, err := ringbuf.NewReader(objs.ProcessLifecycle)
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("create lifecycle reader: %w", err)
	}
	s.reader = reader

	execLink, err := link.Tracepoint("sched", "sched_process_exec", objs.TracepointSchedProcessExec, nil)
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("attach sched_process_exec: %w", err)
	}
	s.links = append(s.links, execLink)

	exitLink, err := link.Tracepoint("sched", "sched_process_exit", objs.TracepointSchedProcessExit, nil)
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("attach sched_process_exit: %w", err)
	}
	s.links = append(s.links, exitLink)

	output := make(chan LifecycleEvent, s.bufferSize)
	go s.read(ctx, output)
	return output, nil
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
		var eventType LifecycleEventType
		switch raw.Type {
		case 1:
			eventType = lifecycleExec
		case 2:
			eventType = lifecycleExit
		default:
			sendLifecycle(ctx, output, LifecycleEvent{Type: lifecycleLoss, Lost: 1, ObservedAt: time.Now()})
			continue
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
		if s.objs != nil {
			closeErr = errors.Join(closeErr, s.objs.Close())
		}
	})
	return closeErr
}

// ensureTracefs attempts to mount tracefs if neither tracefs nor debugfs
// is already available. Tracepoints require one of these filesystems.
// This mirrors what agents like Datadog do automatically rather than
// requiring the user to mount it manually.
func ensureTracefs(logger *zap.Logger) {
	candidates := []string{
		"/sys/kernel/tracing",       // tracefs (preferred, 4.1+)
		"/sys/kernel/debug/tracing", // debugfs fallback
	}
	for _, path := range candidates {
		if _, err := os.Stat(path + "/events"); err == nil {
			return
		}
	}
	target := "/sys/kernel/tracing"
	if err := os.MkdirAll(target, 0o755); err != nil {
		logger.Debug("cannot create tracefs mount point", zap.Error(err))
		return
	}
	if err := syscall.Mount("tracefs", target, "tracefs", 0, ""); err != nil {
		logger.Debug("cannot mount tracefs (needs CAP_SYS_ADMIN)", zap.String("target", target), zap.Error(err))
		return
	}
	logger.Info("mounted tracefs for eBPF tracepoint support", zap.String("path", target))
}

func nextPowerOfTwo(value int) int {
	value--
	for shift := 1; shift < 64; shift <<= 1 {
		value |= value >> shift
	}
	return value + 1
}

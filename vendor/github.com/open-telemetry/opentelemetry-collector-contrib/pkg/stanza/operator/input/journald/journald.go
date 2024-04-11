// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package journald // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/input/journald"

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
)

const waitDuration = 1 * time.Second

func init() {
	operator.Register(operatorType, func() operator.Builder { return NewConfig() })
}

// Build will build a journald input operator from the supplied configuration
func (c Config) Build(logger *zap.SugaredLogger) (operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(logger)
	if err != nil {
		return nil, err
	}

	args, err := c.buildArgs()
	if err != nil {
		return nil, err
	}

	return &Input{
		InputOperator: inputOperator,
		newCmd: func(ctx context.Context, cursor []byte) cmd {
			if cursor != nil {
				args = append(args, "--after-cursor", string(cursor))
			}
			return exec.CommandContext(ctx, "journalctl", args...) // #nosec - ...
			// journalctl is an executable that is required for this operator to function
		},
		json: jsoniter.ConfigFastest,
	}, nil
}

func (c Config) buildArgs() ([]string, error) {
	args := make([]string, 0, 10)

	// Export logs in UTC time
	args = append(args, "--utc")

	// Export logs as JSON
	args = append(args, "--output=json")

	// Continue watching logs until cancelled
	args = append(args, "--follow")

	switch c.StartAt {
	case "end":
	case "beginning":
		args = append(args, "--no-tail")
	default:
		return nil, fmt.Errorf("invalid value '%s' for parameter 'start_at'", c.StartAt)
	}

	for _, unit := range c.Units {
		args = append(args, "--unit", unit)
	}

	for _, identifier := range c.Identifiers {
		args = append(args, "--identifier", identifier)
	}

	args = append(args, "--priority", c.Priority)

	if len(c.Grep) > 0 {
		args = append(args, "--grep", c.Grep)
	}

	if c.Dmesg {
		args = append(args, "--dmesg")
	}

	switch {
	case c.Directory != nil:
		args = append(args, "--directory", *c.Directory)
	case len(c.Files) > 0:
		for _, file := range c.Files {
			args = append(args, "--file", file)
		}
	}

	if len(c.Matches) > 0 {
		matches, err := c.buildMatchesConfig()
		if err != nil {
			return nil, err
		}
		args = append(args, matches...)
	}

	if c.All {
		args = append(args, "--all")
	}

	return args, nil
}

func buildMatchConfig(mc MatchConfig) ([]string, error) {
	re := regexp.MustCompile("^[_A-Z]+$")

	// Sort keys to be consistent with every run and to be predictable for tests
	sortedKeys := make([]string, 0, len(mc))
	for key := range mc {
		if !re.MatchString(key) {
			return []string{}, fmt.Errorf("'%s' is not a valid Systemd field name", key)
		}
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	configs := []string{}
	for _, key := range sortedKeys {
		configs = append(configs, fmt.Sprintf("%s=%s", key, mc[key]))
	}

	return configs, nil
}

func (c Config) buildMatchesConfig() ([]string, error) {
	matches := []string{}

	for i, mc := range c.Matches {
		if i > 0 {
			matches = append(matches, "+")
		}
		mcs, err := buildMatchConfig(mc)
		if err != nil {
			return []string{}, err
		}

		matches = append(matches, mcs...)
	}

	return matches, nil
}

// Input is an operator that process logs using journald
type Input struct {
	helper.InputOperator

	newCmd func(ctx context.Context, cursor []byte) cmd

	persister operator.Persister
	json      jsoniter.API
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

type cmd interface {
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
}

type failedCommand struct {
	err    string
	output string
}

var lastReadCursorKey = "lastReadCursor"

// Start will start generating log entries.
func (operator *Input) Start(persister operator.Persister) error {
	ctx, cancel := context.WithCancel(context.Background())
	operator.cancel = cancel

	// Start from a cursor if there is a saved offset
	cursor, err := persister.Get(ctx, lastReadCursorKey)
	if err != nil {
		return fmt.Errorf("failed to get journalctl state: %w", err)
	}

	operator.persister = persister

	// Start journalctl
	journal := operator.newCmd(ctx, cursor)
	stdout, err := journal.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get journalctl stdout: %w", err)
	}
	stderr, err := journal.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get journalctl stderr: %w", err)
	}
	err = journal.Start()
	if err != nil {
		return fmt.Errorf("start journalctl: %w", err)
	}

	stderrChan := make(chan string)
	failedChan := make(chan failedCommand)

	// Start the wait goroutine
	operator.wg.Add(1)
	go func() {
		defer operator.wg.Done()
		err := journal.Wait()
		message := <-stderrChan

		f := failedCommand{
			output: message,
		}

		if err != nil {
			ee := (&exec.ExitError{})
			if ok := errors.As(err, &ee); ok && ee.ExitCode() != 0 {
				f.err = ee.Error()
			}
		}

		select {
		case failedChan <- f:
		// log an error in case channel is closed
		case <-time.After(waitDuration):
			operator.Logger().Errorw("journalctl command exited", "error", f.err, "output", f.output)
		}
	}()

	// Start the stderr reader goroutine
	operator.wg.Add(1)
	go func() {
		defer operator.wg.Done()

		stderrBuf := bufio.NewReader(stderr)
		messages := []string{}

		for {
			line, err := stderrBuf.ReadBytes('\n')
			if err != nil {
				if !errors.Is(err, io.EOF) {
					operator.Errorw("Received error reading from journalctl stderr", zap.Error(err))
				}
				stderrChan <- strings.Join(messages, "\n")
				return
			}
			messages = append(messages, string(line))
		}
	}()

	// Start the reader goroutine
	operator.wg.Add(1)
	go func() {
		defer operator.wg.Done()

		stdoutBuf := bufio.NewReader(stdout)

		for {
			line, err := stdoutBuf.ReadBytes('\n')
			if err != nil {
				if !errors.Is(err, io.EOF) {
					operator.Errorw("Received error reading from journalctl stdout", zap.Error(err))
				}
				return
			}

			entry, cursor, err := operator.parseJournalEntry(line)
			if err != nil {
				operator.Warnw("Failed to parse journal entry", zap.Error(err))
				continue
			}
			if err := operator.persister.Set(ctx, lastReadCursorKey, []byte(cursor)); err != nil {
				operator.Warnw("Failed to set offset", zap.Error(err))
			}
			operator.Write(ctx, entry)
		}
	}()

	// Wait waitDuration for eventual error
	select {
	case err := <-failedChan:
		if err.err == "" {
			return fmt.Errorf("journalctl command exited")
		}
		return fmt.Errorf("journalctl command failed (%v): %v", err.err, err.output)
	case <-time.After(waitDuration):
		return nil
	}
}

func (operator *Input) parseJournalEntry(line []byte) (*entry.Entry, string, error) {
	var body map[string]any
	err := operator.json.Unmarshal(line, &body)
	if err != nil {
		return nil, "", err
	}

	timestamp, ok := body["__REALTIME_TIMESTAMP"]
	if !ok {
		return nil, "", errors.New("journald body missing __REALTIME_TIMESTAMP field")
	}

	timestampString, ok := timestamp.(string)
	if !ok {
		return nil, "", errors.New("journald field for timestamp is not a string")
	}

	timestampInt, err := strconv.ParseInt(timestampString, 10, 64)
	if err != nil {
		return nil, "", fmt.Errorf("parse timestamp: %w", err)
	}

	delete(body, "__REALTIME_TIMESTAMP")

	cursor, ok := body["__CURSOR"]
	if !ok {
		return nil, "", errors.New("journald body missing __CURSOR field")
	}

	cursorString, ok := cursor.(string)
	if !ok {
		return nil, "", errors.New("journald field for cursor is not a string")
	}

	entry, err := operator.NewEntry(body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create entry: %w", err)
	}

	entry.Timestamp = time.Unix(0, timestampInt*1000) // in microseconds
	return entry, cursorString, nil
}

// Stop will stop generating logs.
func (operator *Input) Stop() error {
	if operator.cancel != nil {
		operator.cancel()
	}
	operator.wg.Wait()
	return nil
}

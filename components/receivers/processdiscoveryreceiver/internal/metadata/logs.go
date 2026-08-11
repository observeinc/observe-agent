package metadata

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
)

type LogsBuilder struct {
	logsBuffer       plog.Logs
	logRecordsBuffer plog.LogRecordSlice
	buildInfo        component.BuildInfo
}

func NewLogsBuilder(settings receiver.Settings) *LogsBuilder {
	return &LogsBuilder{
		logsBuffer:       plog.NewLogs(),
		logRecordsBuffer: plog.NewLogRecordSlice(),
		buildInfo:        settings.BuildInfo,
	}
}

func (lb *LogsBuilder) AppendLogRecord(record plog.LogRecord) {
	record.MoveTo(lb.logRecordsBuffer.AppendEmpty())
}

func (lb *LogsBuilder) Emit() plog.Logs {
	resourceLogs := lb.logsBuffer.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	scopeLogs.Scope().SetName(ScopeName)
	scopeLogs.Scope().SetVersion(lb.buildInfo.Version)
	lb.logRecordsBuffer.MoveAndAppendTo(scopeLogs.LogRecords())
	lb.logRecordsBuffer = plog.NewLogRecordSlice()
	logs := lb.logsBuffer
	lb.logsBuffer = plog.NewLogs()
	return logs
}

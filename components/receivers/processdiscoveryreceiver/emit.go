package processdiscoveryreceiver

import (
	"context"
	"time"

	"github.com/observeinc/observe-agent/components/receivers/processdiscoveryreceiver/internal/metadata"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
)

const discoverySchemaVersion = "2.0-experimental"

type logsEmitter struct {
	receiver *processDiscoveryReceiver
}

func newLogsEmitter(receiver *processDiscoveryReceiver) processEmitter {
	return &logsEmitter{receiver: receiver}
}

func (e *logsEmitter) Emit(ctx context.Context, events []processEvent) error {
	if e.receiver.nextConsumer == nil {
		return nil
	}
	logs := buildLogs(e.receiver.settings, events, e.receiver.resourceAttrs)
	obsCtx := e.receiver.obsrecv.StartLogsOp(ctx)
	err := e.receiver.nextConsumer.ConsumeLogs(obsCtx, logs)
	e.receiver.obsrecv.EndLogsOp(obsCtx, metadata.Type.String(), logs.LogRecordCount(), err)
	return err
}

func buildLogs(settings receiver.Settings, events []processEvent, resourceAttributes map[string]string) plog.Logs {
	builder := metadata.NewLogsBuilder(settings)
	for _, event := range events {
		record := newRecord(event.Process.ObservedAt)
		record.SetEventName(eventName(event.Name))
		record.Body().SetStr("process.discovery")
		attrs := record.Attributes()
		attrs.PutStr("process.discovery.schema.version", discoverySchemaVersion)
		attrs.PutStr("process.discovery.observation.source", event.Process.Source)
		attrs.PutInt("process.pid", int64(event.Process.Key.PID))
		if event.Process.ParentPID > 0 {
			attrs.PutInt("process.parent_pid", int64(event.Process.ParentPID))
		}
		if !event.Process.Key.CreationTime.IsZero() {
			attrs.PutStr("process.creation.time", event.Process.Key.CreationTime.UTC().Format(time.RFC3339Nano))
		}
		putString(attrs, "process.command", event.Process.Command)
		putStrings(attrs, "process.command_args", event.Process.CommandArgs)
		if len(event.Process.CommandArgs) > 0 {
			attrs.PutInt("process.args_count", int64(len(event.Process.CommandArgs)))
		}
		putString(attrs, "process.executable.name", event.Process.ExecutableName)
		putString(attrs, "process.executable.path", event.Process.ExecutablePath)
		putString(attrs, "process.executable.build_id.gnu", event.Process.GNUBuildID)
		putString(attrs, "process.executable.build_id.go", event.Process.GoBuildID)
		putString(attrs, "process.owner", event.Process.Owner)
		if event.Process.UserID > 0 {
			attrs.PutInt("process.user.id", event.Process.UserID)
		}
		putString(attrs, "process.user.name", event.Process.Owner)
		putString(attrs, "process.state", event.Process.State)
		putString(attrs, "process.working_directory", event.Process.WorkingDirectory)
		putString(attrs, "process.linux.cgroup", event.Process.Cgroup)
		putString(attrs, "process.discovery.application.entrypoint", event.Process.ApplicationEntrypoint)
		putString(attrs, "process.executable.arch", event.Process.ExecutableArch)
		if event.Process.GroupLeaderPID > 0 {
			attrs.PutInt("process.group_leader.pid", int64(event.Process.GroupLeaderPID))
		}
		if event.Process.SessionLeaderPID > 0 {
			attrs.PutInt("process.session_leader.pid", int64(event.Process.SessionLeaderPID))
		}
		if event.Process.VirtualPID > 0 {
			attrs.PutInt("process.vpid", int64(event.Process.VirtualPID))
		}
		putString(attrs, "process.runtime.name", event.Process.RuntimeName)
		putString(attrs, "process.runtime.version", event.Process.RuntimeVersion)
		putString(attrs, "process.discovery.runtime.family", event.Process.RuntimeFamily)
		putString(attrs, "process.discovery.runtime.status", event.Process.RuntimeStatus)
		putStrings(attrs, "process.discovery.runtime.assertion", event.Process.RuntimeAssertions)
		putString(attrs, "process.discovery.runtime.version.value", event.Process.InferredVersion)
		putString(attrs, "process.discovery.runtime.version.kind", event.Process.InferredVersionKind)
		putString(attrs, "container.id", event.Process.ContainerID)
		putString(attrs, "container.name", event.Process.ContainerName)
		putString(attrs, "container.runtime", event.Process.ContainerRuntime)
		putString(attrs, "container.image.name", event.Process.ContainerImageName)
		putString(attrs, "container.image.id", event.Process.ContainerImageID)
		putString(attrs, "k8s.namespace.name", event.Process.K8sNamespaceName)
		putString(attrs, "k8s.node.name", event.Process.K8sNodeName)
		putString(attrs, "k8s.pod.uid", event.Process.K8sPodUID)
		putString(attrs, "k8s.pod.name", event.Process.K8sPodName)
		putString(attrs, "k8s.container.name", event.Process.K8sContainerName)
		putString(attrs, "process.discovery.k8s.workload.kind", event.Process.K8sWorkloadKind)
		putString(attrs, "process.discovery.k8s.workload.name", event.Process.K8sWorkloadName)
		putString(attrs, "process.discovery.k8s.workload.uid", event.Process.K8sWorkloadUID)
		putString(attrs, "process.discovery.k8s.correlation.status", event.Process.K8sCorrelationStatus)
		putStrings(attrs, "process.discovery.changed_fields", event.ChangedFields)
		if event.StopReason != "" {
			putString(attrs, "process.discovery.exit.observation_method", event.StopReason)
			attrs.PutStr("process.exit.time", record.Timestamp().AsTime().UTC().Format(time.RFC3339Nano))
		}
		builder.AppendLogRecord(record)
	}
	logs := builder.Emit()
	putResourceAttributes(logs, resourceAttributes)
	return logs
}

func putResourceAttributes(logs plog.Logs, attributes map[string]string) {
	if logs.ResourceLogs().Len() == 0 {
		return
	}
	resource := logs.ResourceLogs().At(0).Resource().Attributes()
	for key, value := range attributes {
		resource.PutStr(key, value)
	}
}

func newRecord(observed time.Time) plog.LogRecord {
	if observed.IsZero() {
		observed = time.Now()
	}
	record := plog.NewLogRecord()
	timestamp := pcommon.NewTimestampFromTime(observed)
	record.SetTimestamp(timestamp)
	record.SetObservedTimestamp(timestamp)
	record.SetSeverityNumber(plog.SeverityNumberInfo)
	record.SetSeverityText("INFO")
	return record
}

func eventName(name string) string {
	switch name {
	case "process.stopped":
		return "process.exited"
	default:
		return name
	}
}

func putString(attrs pcommon.Map, key, value string) {
	if value != "" {
		attrs.PutStr(key, value)
	}
}

func putStrings(attrs pcommon.Map, key string, values []string) {
	if len(values) == 0 {
		return
	}
	slice := attrs.PutEmptySlice(key)
	for _, value := range values {
		slice.AppendEmpty().SetStr(value)
	}
}

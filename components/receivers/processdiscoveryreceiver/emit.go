package processdiscoveryreceiver

import (
	"context"
	"strings"
	"time"

	"github.com/observeinc/observe-agent/components/receivers/processdiscoveryreceiver/internal/metadata"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
)

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
		attrs.PutInt("process.pid", int64(event.Process.Key.PID))
		if event.Process.ParentPID > 0 {
			attrs.PutInt("process.parent_pid", int64(event.Process.ParentPID))
		}
		if !event.Process.Key.CreationTime.IsZero() {
			attrs.PutStr("process.creation.time", event.Process.Key.CreationTime.UTC().Format(time.RFC3339Nano))
		}
		putString(attrs, "process.command", event.Process.Command)
		putStrings(attrs, "process.command_args", event.Process.CommandArgs)
		if event.Process.ArgsCount > 0 {
			attrs.PutInt("process.args_count", int64(event.Process.ArgsCount))
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
		putString(attrs, "process.application.entrypoint", event.Process.ApplicationEntrypoint)
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
		runtimeVersion := event.Process.RuntimeVersion
		if runtimeVersion == "" {
			runtimeVersion = event.Process.InferredVersion
		}
		putString(attrs, "process.runtime.version", runtimeVersion)
		putString(attrs, "process.runtime.family", event.Process.RuntimeFamily)
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
		if event.Process.K8sWorkloadKind != "" && event.Process.K8sWorkloadName != "" {
			prefix := workloadPrefix(event.Process.K8sWorkloadKind)
			putString(attrs, prefix+".name", event.Process.K8sWorkloadName)
			putString(attrs, prefix+".uid", event.Process.K8sWorkloadUID)
		}
		if len(event.Process.OTLPConnections) > 0 {
			connections := attrs.PutEmptySlice("otel.instrumentation.connections")
			for _, conn := range event.Process.OTLPConnections {
				connMap := connections.AppendEmpty().SetEmptyMap()
				connMap.PutStr("remote.host", conn.RemoteHost)
				connMap.PutInt("remote.port", int64(conn.RemotePort))
				connMap.PutStr("network.transport", conn.Transport)
				connMap.PutStr("observed_at", conn.ObservedAt.UTC().Format(time.RFC3339Nano))
				connMap.PutStr("matched_rule", conn.MatchedRule)
			}
		}
		if event.StopReason != "" {
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

func workloadPrefix(kind string) string {
	switch strings.ToLower(kind) {
	case "deployment":
		return "k8s.deployment"
	case "statefulset":
		return "k8s.statefulset"
	case "daemonset":
		return "k8s.daemonset"
	case "replicaset":
		return "k8s.replicaset"
	case "job":
		return "k8s.job"
	case "cronjob":
		return "k8s.cronjob"
	default:
		return "k8s." + strings.ToLower(kind)
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

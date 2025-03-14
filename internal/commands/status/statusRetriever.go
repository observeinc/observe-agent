package status

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/shirou/gopsutil/v3/host"
)

type AgentStatus int64

const (
	NotRunning AgentStatus = iota
	Running    AgentStatus = iota
)

func (s AgentStatus) String() string {
	switch s {
	case NotRunning:
		return "NotRunning"
	case Running:
		return "Running"
	}
	return "Unknown"
}

type StatusData struct {
	Status          string
	OS              string
	Platform        string
	PlatformFamily  string
	PlatformVersion string
	KernelVersion   string
	KernelArch      string
	BootTime        string
	UpTime          string
	HostID          string
	Hostname        string
	AgentMetrics    AgentMetrics
}

type DataTypeStats struct {
	ReceiverAcceptedCount   int
	ReceiverRefusedCount    int
	ExporterSentCount       int
	ExporterSendFailedCount int
}

type AgentMetrics struct {
	ExporterQueueSize     float32
	CPUSeconds            float32
	MemoryUsed            float32
	TotalSysMemory        float32
	Uptime                float32
	AvgServerResponseTime float32
	AvgClientResponseTime float32
	LogsStats             DataTypeStats
	MetricsStats          DataTypeStats
	TracesStats           DataTypeStats
}

func bToMb(b float32) float32 {
	return b / 1024 / 1024
}

func GetAgentStatusFromHealthcheck(baseURL string) (AgentStatus, error) {
	URL := fmt.Sprintf("%s/status", baseURL)
	c := &http.Client{}
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return NotRunning, nil
	}
	resp, err := c.Do(req)
	if err != nil {
		return NotRunning, nil
	}
	if resp.StatusCode == 200 {
		return Running, nil
	} else {
		return NotRunning, nil
	}
}

func getMetricsSum(metrics []*io_prometheus_client.Metric) float64 {
	sum := float64(0)
	for _, metric := range metrics {
		sum += metric.Counter.GetValue()
	}
	return sum
}

func GetAgentMetricsFromEndpoint(baseURL string) (*AgentMetrics, error) {
	URL := fmt.Sprintf("%s/metrics", baseURL)
	c := &http.Client{}
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return nil, err
	}
	var parser expfmt.TextParser
	mf, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, err
	}

	agentMets := AgentMetrics{MetricsStats: DataTypeStats{}, TracesStats: DataTypeStats{}, LogsStats: DataTypeStats{}}
	agentMets.MetricsStats = DataTypeStats{}
	agentMets.TracesStats = DataTypeStats{}
	agentMets.LogsStats = DataTypeStats{}
	for _, v := range mf {
		if v.Type.String() == io_prometheus_client.MetricType_HISTOGRAM.String() {
			met := v.Metric[0]
			switch name := *v.Name; name {
			case "otelcol_http_client_duration":
				agentMets.AvgServerResponseTime = float32(met.Histogram.GetSampleSum()) / float32(met.Histogram.GetSampleCount())
			case "otelcol_http_server_duration":
				agentMets.AvgClientResponseTime = float32(met.Histogram.GetSampleSum()) / float32(met.Histogram.GetSampleCount())
			default:
			}
		} else {
			met := v.Metric[0]
			switch name := *v.Name; name {
			// Log-related metrics
			case "otelcol_receiver_accepted_log_records":
				agentMets.LogsStats.ReceiverAcceptedCount = int(getMetricsSum(v.Metric))
			case "otelcol_receiver_refused_log_records":
				agentMets.LogsStats.ReceiverRefusedCount = int(getMetricsSum(v.Metric))
			case "otelcol_exporter_sent_log_records":
				agentMets.LogsStats.ExporterSentCount = int(getMetricsSum(v.Metric))
			case "otelcol_exporter_send_failed_log_records":
				agentMets.LogsStats.ExporterSendFailedCount = int(getMetricsSum(v.Metric))

			// Metric-related metrics
			case "otelcol_receiver_accepted_metric_points":
				agentMets.MetricsStats.ReceiverAcceptedCount = int(getMetricsSum(v.Metric))
			case "otelcol_receiver_refused_metric_points":
				agentMets.MetricsStats.ReceiverRefusedCount = int(getMetricsSum(v.Metric))
			case "otelcol_exporter_sent_metric_points":
				agentMets.MetricsStats.ExporterSentCount = int(getMetricsSum(v.Metric))
			case "otelcol_exporter_send_failed_metric_points":
				agentMets.MetricsStats.ExporterSendFailedCount = int(getMetricsSum(v.Metric))

			// Trace-related metrics
			case "otelcol_receiver_accepted_spans":
				agentMets.TracesStats.ReceiverAcceptedCount = int(getMetricsSum(v.Metric))
			case "otelcol_receiver_refused_spans":
				agentMets.TracesStats.ReceiverRefusedCount = int(getMetricsSum(v.Metric))
			case "otelcol_exporter_sent_spans":
				agentMets.TracesStats.ExporterSentCount = int(getMetricsSum(v.Metric))
			case "otelcol_exporter_send_failed_spans":
				agentMets.TracesStats.ExporterSendFailedCount = int(getMetricsSum(v.Metric))

			// General metrics
			case "otelcol_exporter_queue_size":
				agentMets.ExporterQueueSize = float32(met.Gauge.GetValue())
			case "otelcol_process_cpu_seconds":
				agentMets.CPUSeconds = float32(getMetricsSum(v.Metric))
			case "otelcol_process_uptime":
				agentMets.Uptime = float32(getMetricsSum(v.Metric))
			case "otelcol_process_memory_rss":
				agentMets.MemoryUsed = bToMb(float32(met.Gauge.GetValue()))
			case "otelcol_process_runtime_total_sys_memory_bytes":
				agentMets.TotalSysMemory = bToMb(float32(met.Gauge.GetValue()))
			default:
			}
		}
	}
	return &agentMets, nil
}

func GetStatusData() (*StatusData, error) {
	agentMets, err := GetAgentMetricsFromEndpoint("http://localhost:8888")
	if err != nil {
		fmt.Println("Error getting agent metrics: ", err)
		agentMets = &AgentMetrics{}
	}
	hostInfo, err := host.Info()
	if err != nil {
		hostInfo = &host.InfoStat{}
	}
	hn, err := os.Hostname()
	if err != nil {
		hn = "unknown"
	}
	bt := time.Unix(int64(hostInfo.BootTime), 0)
	uptime, err := time.ParseDuration(strconv.FormatUint(hostInfo.Uptime, 10) + "s")
	if err != nil {
		uptime = time.Duration(0)
	}
	status, err := GetAgentStatusFromHealthcheck("http://localhost:13133")
	if err != nil {
		status = NotRunning
	}

	data := StatusData{
		Status:          status.String(),
		BootTime:        bt.Format(time.RFC3339),
		UpTime:          uptime.Round(time.Second).String(),
		HostID:          hostInfo.HostID,
		Hostname:        hn,
		OS:              hostInfo.OS,
		Platform:        hostInfo.Platform,
		PlatformFamily:  hostInfo.PlatformFamily,
		PlatformVersion: hostInfo.PlatformVersion,
		KernelVersion:   hostInfo.KernelVersion,
		KernelArch:      hostInfo.KernelArch,
		AgentMetrics:    *agentMets,
	}
	return &data, nil
}

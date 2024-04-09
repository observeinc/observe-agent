package status

import (
	_ "embed"
	"net/http"
	"net/http/httptest"
	"testing"
)

//go:embed testfixtures/testmetrics
var testmetrics []byte

func TestGetAgentStatusFromHealthcheck(t *testing.T) {
	var tests = []struct {
		name         string
		responseCode int
		want         AgentStatus
	}{
		{
			name:         "Negative Case",
			responseCode: 404,
			want:         NotRunning,
		},
		{
			name:         "Positive Case",
			responseCode: 200,
			want:         Running,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseCode)
			}))
			defer server.Close()

			status, err := GetAgentStatusFromHealthcheck(server.URL)
			if err != nil {
				t.Error(err)
			}
			if status != tt.want {
				t.Errorf("Expected %s, got %d", tt.want, status)
			}
		})
	}
}

func TestGetAgentMetricsFromEndpoint(t *testing.T) {
	var tests = []struct {
		name         string
		responseCode int
		responseBody []byte
		want         *AgentMetrics
	}{
		{
			name:         "Negative Case",
			responseCode: 404,
			responseBody: []byte(""),
			want:         nil,
		},
		{
			name:         "Positive Case",
			responseCode: 200,
			responseBody: testmetrics,
			want: &AgentMetrics{
				ExporterQueueSize:     0,
				CPUSeconds:            2.6785486,
				MemoryUsed:            49.546875,
				TotalSysMemory:        33.784195,
				Uptime:                43.018906,
				AvgServerResponseTime: 32.50218,
				AvgClientResponseTime: 18.633617,
				MetricsStats: DataTypeStats{
					ReceiverAcceptedCount:   1995,
					ReceiverRefusedCount:    0,
					ExporterSentCount:       1995,
					ExporterSendFailedCount: 0,
				},
				TracesStats: DataTypeStats{
					ReceiverAcceptedCount:   5126,
					ReceiverRefusedCount:    0,
					ExporterSentCount:       5126,
					ExporterSendFailedCount: 0,
				},
				LogsStats: DataTypeStats{
					ReceiverAcceptedCount:   328807,
					ReceiverRefusedCount:    0,
					ExporterSentCount:       328799,
					ExporterSendFailedCount: 0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseCode)
				w.Write(tt.responseBody)
			}))
			defer server.Close()

			agentMets, err := GetAgentMetricsFromEndpoint(server.URL)
			if err != nil {
				t.Error(err)
			}
			if tt.want != nil {
				if *agentMets != *tt.want {
					t.Errorf("Expected %#v, got %#v", tt.want, agentMets)
				}

			}
		})
	}
}

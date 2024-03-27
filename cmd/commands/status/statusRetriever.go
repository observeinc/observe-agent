package status

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

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
}

func GetAgentStatusFromHealthcheck() (AgentStatus, error) {
	c := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:13133/status", nil)
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

func GetStatusData() (*StatusData, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return nil, err
	}
	hn, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	bt := time.Unix(int64(hostInfo.BootTime), 0)
	uptime, err := time.ParseDuration(strconv.FormatUint(hostInfo.Uptime, 10) + "s")
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	status, err := GetAgentStatusFromHealthcheck()
	if err != nil {
		return nil, err
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
	}
	return &data, nil
}

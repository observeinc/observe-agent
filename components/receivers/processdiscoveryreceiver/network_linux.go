//go:build linux

package processdiscoveryreceiver

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func discoverNetworkEndpoints(procRoot string, pid int32, maxPorts int) ([]NetworkEndpoint, string) {
	inodes, err := processSocketInodes(procRoot, pid)
	if err != nil {
		return nil, networkEvidenceStatus(err)
	}
	counts := make(map[NetworkEndpoint]int64)
	openedTables := 0
	complete := true
	for _, table := range []struct {
		name, networkType string
	}{{"tcp", "ipv4"}, {"tcp6", "ipv6"}} {
		file, openErr := os.Open(filepath.Join(procRoot, strconv.Itoa(int(pid)), "net", table.name))
		if openErr != nil {
			complete = false
			continue
		}
		openedTables++
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 10 || fields[0] == "sl" {
				continue
			}
			state := tcpConnectionState(fields[3])
			if state == "" {
				continue
			}
			if _, ok := inodes[fields[9]]; !ok {
				continue
			}
			port, parseErr := parseProcNetPort(fields[1])
			if parseErr != nil {
				continue
			}
			key := NetworkEndpoint{LocalPort: port, Type: table.networkType, State: state}
			counts[key]++
		}
		if scanner.Err() != nil {
			complete = false
		}
		_ = file.Close()
	}
	endpoints := make([]NetworkEndpoint, 0, len(counts))
	for endpoint, count := range counts {
		endpoint.Count = count
		endpoints = append(endpoints, endpoint)
	}
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].LocalPort != endpoints[j].LocalPort {
			return endpoints[i].LocalPort < endpoints[j].LocalPort
		}
		return endpoints[i].Type < endpoints[j].Type
	})
	if len(endpoints) > maxPorts {
		endpoints = endpoints[:maxPorts]
	}
	if openedTables == 0 {
		return endpoints, "unavailable"
	}
	if !complete {
		return endpoints, "partial"
	}
	return endpoints, "complete"
}

func (s *procFSSource) ResolveAcceptedConnection(pid, fd int32) (NetworkEndpoint, error) {
	target, err := os.Readlink(filepath.Join(s.root, strconv.Itoa(int(pid)), "fd", strconv.Itoa(int(fd))))
	if err != nil {
		return NetworkEndpoint{}, err
	}
	inode := socketInode(target)
	if inode == "" {
		return NetworkEndpoint{}, fmt.Errorf("fd %d is not a socket", fd)
	}
	for _, table := range []struct {
		name, networkType string
	}{{"tcp", "ipv4"}, {"tcp6", "ipv6"}} {
		file, openErr := os.Open(filepath.Join(s.root, strconv.Itoa(int(pid)), "net", table.name))
		if openErr != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 10 || fields[9] != inode {
				continue
			}
			port, parseErr := parseProcNetPort(fields[1])
			_ = file.Close()
			if parseErr != nil {
				return NetworkEndpoint{}, parseErr
			}
			return NetworkEndpoint{LocalPort: port, Type: table.networkType, Count: 1}, nil
		}
		_ = file.Close()
	}
	return NetworkEndpoint{}, fmt.Errorf("socket inode %s is no longer available", inode)
}

func processSocketInodes(procRoot string, pid int32) (map[string]struct{}, error) {
	entries, err := os.ReadDir(filepath.Join(procRoot, strconv.Itoa(int(pid)), "fd"))
	if err != nil {
		return nil, err
	}
	inodes := make(map[string]struct{})
	for _, entry := range entries {
		target, readErr := os.Readlink(filepath.Join(procRoot, strconv.Itoa(int(pid)), "fd", entry.Name()))
		if readErr == nil {
			if inode := socketInode(target); inode != "" {
				inodes[inode] = struct{}{}
			}
		}
	}
	return inodes, nil
}

func socketInode(target string) string {
	if strings.HasPrefix(target, "socket:[") && strings.HasSuffix(target, "]") {
		return strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]")
	}
	return ""
}

func parseProcNetPort(localAddress string) (int, error) {
	index := strings.LastIndex(localAddress, ":")
	if index < 0 {
		return 0, fmt.Errorf("invalid local address %q", localAddress)
	}
	port, err := strconv.ParseUint(localAddress[index+1:], 16, 16)
	return int(port), err
}

func tcpConnectionState(value string) string {
	switch value {
	case "01":
		return "established"
	case "0A":
		return "listen"
	default:
		return ""
	}
}

func networkEvidenceStatus(err error) string {
	if os.IsPermission(err) {
		return "inaccessible"
	}
	if os.IsNotExist(err) {
		return "unavailable"
	}
	return "partial"
}

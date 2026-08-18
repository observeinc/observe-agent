//go:build linux

package processdiscoveryreceiver

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type otlpEndpoint struct {
	Host string
	Port int
}

var standardOTLPPorts = []int{4317, 4318}

func detectOTLPConnections(procRoot string, pid int32, endpoints []otlpEndpoint, maxConnections int) InstrumentationEvidence {
	inodes, err := processSocketInodes(procRoot, pid)
	if err != nil {
		return InstrumentationEvidence{Status: evidenceStatus(err)}
	}

	var connections []OTLPConnection
	now := time.Now()
	openedTables := 0

	type connKey struct {
		host      string
		port      int
		transport string
	}
	seen := make(map[connKey]struct{})

	for _, table := range []struct {
		name      string
		transport string
		ipLen     int
	}{
		{"tcp", "tcp4", 4},
		{"tcp6", "tcp6", 16},
	} {
		file, openErr := os.Open(filepath.Join(procRoot, strconv.Itoa(int(pid)), "net", table.name))
		if openErr != nil {
			continue
		}
		openedTables++
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			if len(fields) < 10 || fields[0] == "sl" {
				continue
			}
			// Only ESTABLISHED connections (state 01)
			if fields[3] != "01" {
				continue
			}
			if _, ok := inodes[fields[9]]; !ok {
				continue
			}
			remoteIP, remotePort, parseErr := parseProcNetAddress(fields[2], table.ipLen)
			if parseErr != nil {
				continue
			}
			if rule := matchOTLPEndpoint(remoteIP, remotePort, endpoints); rule != "" {
				key := connKey{host: remoteIP, port: remotePort, transport: table.transport}
				if _, dup := seen[key]; dup {
					continue
				}
				seen[key] = struct{}{}
				connections = append(connections, OTLPConnection{
					RemoteHost:  remoteIP,
					RemotePort:  remotePort,
					Transport:   table.transport,
					ObservedAt:  now,
					MatchedRule: rule,
				})
				if len(connections) >= maxConnections {
					_ = file.Close()
					return InstrumentationEvidence{OTLPConnections: connections, Status: "detected"}
				}
			}
		}
		_ = file.Close()
	}

	if openedTables == 0 {
		return InstrumentationEvidence{Status: "unavailable"}
	}
	if len(connections) > 0 {
		return InstrumentationEvidence{OTLPConnections: connections, Status: "detected"}
	}
	return InstrumentationEvidence{Status: "none_detected"}
}

func matchOTLPEndpoint(remoteIP string, remotePort int, endpoints []otlpEndpoint) string {
	for _, ep := range endpoints {
		if ep.Port == remotePort && matchesHost(remoteIP, ep.Host) {
			return fmt.Sprintf("configured:%s:%d", ep.Host, ep.Port)
		}
	}
	for _, port := range standardOTLPPorts {
		if remotePort == port {
			return fmt.Sprintf("standard_port:%d", port)
		}
	}
	return ""
}

func matchesHost(remoteIP, host string) bool {
	if host == remoteIP {
		return true
	}
	addrs, err := net.LookupHost(host)
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		if addr == remoteIP {
			return true
		}
	}
	return false
}

func parseProcNetAddress(hexAddr string, ipLen int) (string, int, error) {
	parts := strings.SplitN(hexAddr, ":", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid address %q", hexAddr)
	}
	port, err := strconv.ParseUint(parts[1], 16, 16)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port in %q: %w", hexAddr, err)
	}
	ipHex := parts[0]
	ipBytes, err := hex.DecodeString(ipHex)
	if err != nil {
		return "", 0, fmt.Errorf("invalid IP hex in %q: %w", hexAddr, err)
	}
	var ip net.IP
	if ipLen == 4 {
		if len(ipBytes) != 4 {
			return "", 0, fmt.Errorf("expected 4 bytes for IPv4, got %d", len(ipBytes))
		}
		// /proc/net/tcp stores IPv4 in little-endian 32-bit word
		ip = net.IPv4(ipBytes[3], ipBytes[2], ipBytes[1], ipBytes[0])
	} else {
		if len(ipBytes) != 16 {
			return "", 0, fmt.Errorf("expected 16 bytes for IPv6, got %d", len(ipBytes))
		}
		// /proc/net/tcp6 stores IPv6 as four 32-bit words in host byte order (little-endian on x86)
		ip = make(net.IP, 16)
		for i := 0; i < 4; i++ {
			word := binary.LittleEndian.Uint32(ipBytes[i*4 : (i+1)*4])
			binary.BigEndian.PutUint32(ip[i*4:(i+1)*4], word)
		}
	}
	return ip.String(), int(port), nil
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

func evidenceStatus(err error) string {
	if os.IsPermission(err) {
		return "inaccessible"
	}
	if os.IsNotExist(err) {
		return "unavailable"
	}
	return "unavailable"
}

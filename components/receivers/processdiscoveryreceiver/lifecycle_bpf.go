//go:build linux

package processdiscoveryreceiver

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -no-strip -target bpfel lifecycle bpf/lifecycle.bpf.c -- -I bpf -Wall -O2 -g

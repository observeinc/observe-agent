package gocommonscripts

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type Process struct {
	pid int
}
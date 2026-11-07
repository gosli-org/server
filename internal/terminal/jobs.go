package terminal

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// jobGroups snapshots descendants before their shell exits and they are
// reparented. POSIX ps keeps this independent of Linux-specific /proc files.
func jobGroups(sessions []*Session) (map[int]struct{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "ps", "-A", "-o", "pid=", "-o", "ppid=", "-o", "pgid=").Output()
	if err != nil {
		return nil, fmt.Errorf("list terminal jobs: %w", err)
	}
	type process struct{ pid, group int }
	children := make(map[int][]process)
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, fmt.Errorf("list terminal jobs: invalid process row %q", line)
		}
		ids := [3]int{}
		for i, field := range fields {
			ids[i], err = strconv.Atoi(field)
			if err != nil {
				return nil, fmt.Errorf("list terminal jobs: %w", err)
			}
		}
		children[ids[1]] = append(children[ids[1]], process{pid: ids[0], group: ids[2]})
	}
	groups := make(map[int]struct{})
	for _, s := range sessions {
		queue := []int{s.cmd.Process.Pid}
		for len(queue) != 0 {
			pid := queue[0]
			queue = queue[1:]
			for _, child := range children[pid] {
				if child.group > 0 && child.group != s.cmd.Process.Pid {
					groups[child.group] = struct{}{}
				}
				queue = append(queue, child.pid)
			}
		}
	}
	return groups, nil
}

// waitGroups lets cooperative jobs exit and lets bash reap terminated jobs.
// The shared deadline bounds shutdown independently of the number of jobs.
func waitGroups(groups map[int]struct{}) {
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for len(groups) != 0 {
		for group := range groups {
			if syscall.Kill(-group, 0) == syscall.ESRCH {
				delete(groups, group)
			}
		}
		if len(groups) == 0 {
			return
		}
		select {
		case <-deadline.C:
			return
		case <-tick.C:
		}
	}
}

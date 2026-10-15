package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Run struct {
	At     string `json:"at"`
	Status string `json:"status"`
	Output string `json:"output,omitempty"`
}

type Job struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Cron       string            `json:"cron"`
	Once       bool              `json:"once"`
	RunAt      string            `json:"run_at,omitempty"`
	Enabled    bool              `json:"enabled"`
	TimeoutSec int               `json:"timeout_sec"`
	Payload    map[string]string `json:"payload"`
	Missed     bool              `json:"missed"`
	LastSlot   string            `json:"last_slot,omitempty"`
	Runs       []Run             `json:"runs"`
}

type Handler func(context.Context, Job) (string, error)

type Scheduler struct {
	path     string
	dataDir  string
	mu       sync.Mutex
	jobs     []Job
	handlers map[string]Handler
	stop     chan struct{}
}

func Open(dataDir string) (*Scheduler, error) {
	dir := filepath.Join(dataDir, "cron")
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		return nil, err
	}
	s := &Scheduler{
		path: filepath.Join(dir, "jobs.json"), dataDir: dataDir,
		handlers: map[string]Handler{}, stop: make(chan struct{}),
	}
	b, err := os.ReadFile(s.path)
	if err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &s.jobs)
	}
	if s.jobs == nil {
		s.jobs = []Job{}
	}
	s.markMissed(time.Now())
	return s, nil
}

func (s *Scheduler) Register(kind string, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[kind] = handler
}

func (s *Scheduler) Start() {
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.stop:
				return
			case now := <-ticker.C:
				s.tick(now)
			}
		}
	}()
}

func (s *Scheduler) Stop() { close(s.stop) }

// Reload reads jobs from disk and marks overdue one-shot runs as missed without executing them.
func (s *Scheduler) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.jobs = []Job{}
			return nil
		}
		return err
	}
	var jobs []Job
	if len(b) > 0 {
		if err := json.Unmarshal(b, &jobs); err != nil {
			return err
		}
	}
	if jobs == nil {
		jobs = []Job{}
	}
	s.jobs = jobs
	s.markMissed(time.Now())
	return nil
}

func (s *Scheduler) List() []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Job{}, s.jobs...)
	return out
}

func (s *Scheduler) Create(job Job) (Job, error) {
	if job.Type != "script" && !Valid(job.Cron) && !job.Once {
		return Job{}, fmt.Errorf("计划表达式须为五段 cron")
	}
	if job.Cron != "" && !Valid(job.Cron) {
		return Job{}, fmt.Errorf("计划表达式须为五段 cron")
	}
	if job.ID == "" {
		job.ID = fmt.Sprintf("job-%d", time.Now().UnixNano())
	}
	if job.Payload == nil {
		job.Payload = map[string]string{}
	}
	if job.Runs == nil {
		job.Runs = []Run{}
	}
	job.Enabled = true
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append(s.jobs, job)
	return job, s.save()
}

func (s *Scheduler) SetEnabled(id string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.jobs {
		if s.jobs[i].ID == id {
			s.jobs[i].Enabled = enabled
			return s.save()
		}
	}
	return fmt.Errorf("任务不存在")
}

func (s *Scheduler) RunNow(ctx context.Context, id string) (Run, error) {
	s.mu.Lock()
	var job Job
	var idx int
	found := false
	for i := range s.jobs {
		if s.jobs[i].ID == id {
			job = s.jobs[i]
			idx = i
			found = true
		}
	}
	handler := s.handlers[job.Type]
	s.mu.Unlock()
	if !found {
		return Run{}, fmt.Errorf("任务不存在")
	}
	run := s.execute(ctx, job, handler)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[idx].Runs = append(s.jobs[idx].Runs, run)
	return run, s.save()
}

func (s *Scheduler) execute(ctx context.Context, job Job, handler Handler) Run {
	timeout := job.TimeoutSec
	if timeout <= 0 {
		timeout = 30
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	run := Run{At: time.Now().UTC().Format(time.RFC3339), Status: "success"}
	if job.Type == "script" {
		out, err := runScript(runCtx, s.dataDir, job.Payload["script"], time.Duration(timeout)*time.Second)
		run.Output = out
		if err != nil {
			run.Status = "failed"
			if runCtx.Err() != nil {
				run.Status = "timeout"
			}
			run.Output = strings.TrimSpace(out + " " + err.Error())
		}
		return run
	}
	if handler == nil {
		run.Status = "failed"
		run.Output = "没有对应的任务处理器"
		return run
	}
	out, err := handler(runCtx, job)
	run.Output = out
	if err != nil {
		run.Status = "failed"
		run.Output = err.Error()
	}
	return run
}

func runScript(ctx context.Context, dataDir, name string, timeout time.Duration) (string, error) {
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		return "", fmt.Errorf("脚本必须放在面板脚本目录")
	}
	path := filepath.Join(dataDir, "cron", "scripts", name)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("脚本不存在")
	}
	cmd := exec.CommandContext(ctx, "sh", path)
	cmd.Dir = filepath.Join(dataDir, "cron", "scripts")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		return cmd.Process.Kill()
	}
	cmd.WaitDelay = time.Second
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(out), fmt.Errorf("脚本超时（%s）", timeout)
	}
	return string(out), err
}

func minuteSlot(now time.Time) string {
	return now.Format("2006-01-02T15:04")
}

func (s *Scheduler) tick(now time.Time) {
	s.mu.Lock()
	type pair struct {
		idx     int
		job     Job
		handler Handler
	}
	slot := minuteSlot(now)
	var due []pair
	for i := range s.jobs {
		job := s.jobs[i]
		if !job.Enabled {
			continue
		}
		if job.Once {
			if job.Missed || len(job.Runs) > 0 {
				continue
			}
			at, err := time.Parse(time.RFC3339, job.RunAt)
			if err != nil || now.Before(at) || at.Before(now.Add(-time.Minute)) {
				continue
			}
		} else if !Due(job.Cron, now) || job.LastSlot == slot {
			continue
		} else {
			s.jobs[i].LastSlot = slot
			job.LastSlot = slot
		}
		due = append(due, pair{i, job, s.handlers[job.Type]})
	}
	if len(due) > 0 {
		_ = s.save()
	}
	s.mu.Unlock()
	for _, item := range due {
		run := s.execute(context.Background(), item.job, item.handler)
		s.mu.Lock()
		if item.idx < len(s.jobs) && s.jobs[item.idx].ID == item.job.ID {
			s.jobs[item.idx].Runs = append(s.jobs[item.idx].Runs, run)
			_ = s.save()
		}
		s.mu.Unlock()
	}
}

func (s *Scheduler) markMissed(now time.Time) {
	changed := false
	for i := range s.jobs {
		job := &s.jobs[i]
		if !job.Once || job.Missed || len(job.Runs) > 0 {
			continue
		}
		at, err := time.Parse(time.RFC3339, job.RunAt)
		if err != nil || !at.Before(now.Add(-time.Minute)) {
			continue
		}
		job.Missed = true
		changed = true
	}
	if changed {
		_ = s.save()
	}
}

func (s *Scheduler) save() error {
	b, err := json.MarshalIndent(s.jobs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o644)
}

func Valid(expr string) bool {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return false
	}
	for _, field := range fields {
		if field == "*" {
			continue
		}
		if _, err := strconv.Atoi(field); err != nil {
			return false
		}
	}
	return true
}

func Due(expr string, now time.Time) bool {
	if !Valid(expr) {
		return false
	}
	fields := strings.Fields(expr)
	parts := []int{now.Minute(), now.Hour(), now.Day(), int(now.Month()), int(now.Weekday())}
	for i, field := range fields {
		if field == "*" {
			continue
		}
		n, _ := strconv.Atoi(field)
		if n != parts[i] {
			return false
		}
	}
	return true
}

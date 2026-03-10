package dispatch

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aliencodez/botdo/internal/agent"
	"github.com/aliencodez/botdo/internal/logstore"
	"github.com/aliencodez/botdo/internal/model"
	"github.com/aliencodez/botdo/internal/store"
)

// Dispatcher polls for pending tasks and executes them via registered adapters.
type Dispatcher struct {
	store    store.Store
	logs     logstore.LogStore
	adapters map[string]agent.Adapter
	workDir  string
	interval time.Duration
	inflight sync.Map // taskID int64 -> struct{}
}

// New creates a Dispatcher. interval defaults to 5s if zero.
func New(s store.Store, ls logstore.LogStore, workDir string, interval time.Duration, adapters ...agent.Adapter) *Dispatcher {
	if interval == 0 {
		interval = 5 * time.Second
	}
	d := &Dispatcher{
		store:    s,
		logs:     ls,
		adapters: make(map[string]agent.Adapter, len(adapters)),
		workDir:  workDir,
		interval: interval,
	}
	for _, a := range adapters {
		d.adapters[a.AgentType()] = a
	}
	return d
}

// Start begins the polling loop. It blocks until ctx is cancelled.
func (d *Dispatcher) Start(ctx context.Context) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.poll(ctx)
		}
	}
}

func (d *Dispatcher) poll(ctx context.Context) {
	tasks, err := d.store.ListTasks(store.Filter{Status: string(model.StatusPending)})
	if err != nil {
		log.Printf("dispatch: list tasks: %v", err)
		return
	}
	for _, t := range tasks {
		if _, ok := d.adapters[string(t.Agent)]; !ok {
			continue
		}
		if _, loaded := d.inflight.LoadOrStore(t.ID, struct{}{}); loaded {
			continue // already running
		}
		// Claim the task before launching goroutine.
		t.Status = model.StatusInProgress
		if err := d.store.UpdateTask(t); err != nil {
			log.Printf("dispatch: claim task %d: %v", t.ID, err)
			d.inflight.Delete(t.ID)
			continue
		}
		go d.run(ctx, t)
	}
}

func (d *Dispatcher) run(ctx context.Context, t *model.Task) {
	defer d.inflight.Delete(t.ID)

	adapter := d.adapters[string(t.Agent)]
	lw := d.logs.Writer(t.ID)
	prompt := buildPrompt(t)

	workDir := d.workDir
	if t.ProjectID != nil {
		if proj, perr := d.store.GetProject(*t.ProjectID); perr == nil && proj.WorkDir != "" {
			workDir = proj.WorkDir
		}
	}

	lw.WriteLine(fmt.Sprintf("[dispatch] starting task %d: %s", t.ID, t.Title))

	err := adapter.Run(ctx, t.ID, prompt, workDir, t.Permissions, lw)

	// Re-fetch task to get latest state before updating.
	latest, ferr := d.store.GetTask(t.ID)
	if ferr != nil {
		log.Printf("dispatch: get task %d after run: %v", t.ID, ferr)
		return
	}
	if err != nil {
		lw.WriteLine(fmt.Sprintf("[dispatch] task %d failed: %v", t.ID, err))
		latest.Status = model.StatusFailed
	} else {
		lw.WriteLine(fmt.Sprintf("[dispatch] task %d done", t.ID))
		latest.Status = model.StatusDone
	}
	if uerr := d.store.UpdateTask(latest); uerr != nil {
		log.Printf("dispatch: update task %d status: %v", t.ID, uerr)
	}
}

func buildPrompt(t *model.Task) string {
	if t.Description == "" {
		return t.Title
	}
	return t.Title + "\n\n" + t.Description
}

package selfdestruct

import (
	"sync"
	"time"
)

type Policy struct {
	KillDate    *time.Time    `json:"kill_date"`
	MaxBeacons  int64         `json:"max_beacons"`
	MaxIdle     time.Duration `json:"max_idle"`
	EraseOnKill bool          `json:"erase_on_kill"`
}

type Controller struct {
	mu          sync.Mutex
	policy      Policy
	beaconCount int64
	lastTask    time.Time
	armed       bool
	activated   bool
	onDestroy   func(reason string)
	onTask      func()
}

func New(p Policy, destroy func(reason string)) *Controller {
	return &Controller{
		policy:    p,
		lastTask:  time.Now(),
		onDestroy: destroy,
	}
}

func (c *Controller) Arm() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.armed = true
}

func (c *Controller) Disarm() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.armed = false
}

func (c *Controller) RecordBeacon() {
	c.mu.Lock()
	if !c.armed {
		c.mu.Unlock()
		return
	}
	c.beaconCount++
	if c.policy.MaxBeacons > 0 && c.beaconCount >= c.policy.MaxBeacons {
		c.activateLocked("max_beacons")
		c.mu.Unlock()
		return
	}
	if c.policy.KillDate != nil && time.Now().After(*c.policy.KillDate) {
		c.activateLocked("kill_date_reached")
		c.mu.Unlock()
		return
	}
	now := time.Now()
	if c.policy.MaxIdle > 0 && now.Sub(c.lastTask) > c.policy.MaxIdle {
		c.activateLocked("max_idle")
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
}

func (c *Controller) RecordTask() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastTask = time.Now()
	if c.onTask != nil {
		c.onTask()
	}
}

func (c *Controller) Kill() {
	c.mu.Lock()
	c.activateLocked("command_kill")
	c.mu.Unlock()
}

func (c *Controller) IsActivated() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.activated
}

func (c *Controller) ShouldDestroy() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.activated
}

func (c *Controller) Stats() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return map[string]interface{}{
		"beacons":      c.beaconCount,
		"armed":        c.armed,
		"activated":    c.activated,
		"last_task_at": c.lastTask,
	}
}

func (c *Controller) activateLocked(reason string) {
	if c.activated {
		return
	}
	c.activated = true
	if c.onDestroy != nil {
		c.onDestroy(reason)
	}
}

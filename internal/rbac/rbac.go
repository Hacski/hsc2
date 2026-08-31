package rbac

import (
	"fmt"
	"sort"
	"sync"
)

type Permission string

const (
	PermViewSessions    Permission = "sessions.view"
	PermControlSessions Permission = "sessions.control"
	PermKillSessions    Permission = "sessions.kill"
	PermTransferFiles   Permission = "files.transfer"
	PermRunExec         Permission = "exec.run"
	PermManageUsers     Permission = "users.manage"
	PermManageListners  Permission = "listeners.manage"
	PermViewAudit       Permission = "audit.view"
)

type Role string

const (
	RoleViewer   Role = "viewer"
	RoleOperator Role = "operator"
	RoleAdmin    Role = "admin"
)

type roleDef struct {
	perms map[Permission]bool
}

var roleTable = map[Role]roleDef{
	RoleViewer: {
		perms: map[Permission]bool{
			PermViewSessions: true,
			PermViewAudit:    true,
		},
	},
	RoleOperator: {
		perms: map[Permission]bool{
			PermViewSessions:    true,
			PermControlSessions: true,
			PermTransferFiles:   true,
			PermRunExec:         true,
			PermViewAudit:       true,
		},
	},
	RoleAdmin: {
		perms: map[Permission]bool{
			PermViewSessions:    true,
			PermControlSessions: true,
			PermKillSessions:    true,
			PermTransferFiles:   true,
			PermRunExec:         true,
			PermManageUsers:     true,
			PermManageListners:  true,
			PermViewAudit:       true,
		},
	},
}

type Principal struct {
	Username string
	Role     Role
	Scopes   map[string]bool
}

func (p Principal) HasPermission(perm Permission) bool {
	def, ok := roleTable[p.Role]
	if !ok {
		return false
	}
	return def.perms[perm]
}

func (p Principal) HasScope(scope string) bool {
	return p.Scopes[scope]
}

type Authorizer struct {
	mu        sync.RWMutex
	overrides map[string]map[string][]Permission
}

func NewAuthorizer() *Authorizer {
	return &Authorizer{overrides: map[string]map[string][]Permission{}}
}

func (a *Authorizer) Allowed(p Principal, perm Permission, scope string) error {
	if scope != "" && !p.Scopes[scope] {
		return fmt.Errorf("principal %s lacks scope %s", p.Username, scope)
	}
	if p.HasPermission(perm) {
		return nil
	}
	return fmt.Errorf("denied: %s lacks permission %s", p.Username, perm)
}

func (a *Authorizer) Roles() []string {
	out := []string{string(RoleViewer), string(RoleOperator), string(RoleAdmin)}
	sort.Strings(out)
	return out
}

func (a *Authorizer) PermissionsFor(role Role) []Permission {
	def := roleTable[role]
	out := []Permission{}
	for p := range def.perms {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (a *Authorizer) AddOverrides(username string, scope string, perms []Permission) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.overrides[username] == nil {
		a.overrides[username] = map[string][]Permission{}
	}
	a.overrides[username][scope] = perms
}

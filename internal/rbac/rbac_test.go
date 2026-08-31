package rbac

import "testing"

func TestRolePermissions(t *testing.T) {
	viewer := Principal{Username: "v", Role: RoleViewer, Scopes: map[string]bool{"eng1": true}}
	if !viewer.HasPermission(PermViewSessions) {
		t.Fatal("viewer must view sessions")
	}
	if viewer.HasPermission(PermKillSessions) {
		t.Fatal("viewer must not kill sessions")
	}
}

func TestAdminFullAccess(t *testing.T) {
	admin := Principal{Username: "a", Role: RoleAdmin}
	for _, p := range []Permission{
		PermViewSessions, PermControlSessions, PermKillSessions,
		PermTransferFiles, PermRunExec, PermManageUsers, PermManageListners, PermViewAudit,
	} {
		if !admin.HasPermission(p) {
			t.Fatalf("admin lacks %s", p)
		}
	}
}

func TestScopeEnforcement(t *testing.T) {
	auth := NewAuthorizer()
	op := Principal{Username: "op", Role: RoleOperator, Scopes: map[string]bool{"eng1": true}}
	if err := auth.Allowed(op, PermRunExec, "eng1"); err != nil {
		t.Fatal(err)
	}
	if err := auth.Allowed(op, PermRunExec, "eng2"); err == nil {
		t.Fatal("must be denied outside scope")
	}
}

func TestDenyOutsideRole(t *testing.T) {
	auth := NewAuthorizer()
	viewer := Principal{Username: "v", Role: RoleViewer, Scopes: map[string]bool{"eng1": true}}
	if err := auth.Allowed(viewer, PermKillSessions, "eng1"); err == nil {
		t.Fatal("viewer must be denied kill")
	}
}

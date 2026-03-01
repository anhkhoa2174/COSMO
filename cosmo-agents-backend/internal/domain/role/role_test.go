package role

import (
	"github.com/google/uuid"
	"testing"
)

func TestRoleDefaults(t *testing.T) {
	role := Role{}

	if role.TableName() != "roles" {
		t.Fatalf("expected table name roles")
	}

	if role.Name != "" {
		t.Fatalf("expected zero value name before hooks, got %s", role.Name)
	}

	role.Name = RoleNameAdmin
	role.Status = RoleStatusActive
	role.UserID = uuid.New()
	role.OrganizationID = uuid.New()

	if role.Name != RoleNameAdmin {
		t.Fatalf("expected name admin")
	}
	if role.Status != RoleStatusActive {
		t.Fatalf("expected status active")
	}
}

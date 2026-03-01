package inbound_lead_form

import (
	"testing"

	"github.com/google/uuid"
)

func TestInboundLeadFormTableName(t *testing.T) {
	if (InboundLeadForm{}).TableName() != "inbound_lead_forms" {
		t.Fatalf("unexpected table name")
	}
}

func TestFormFieldDefaults(t *testing.T) {
	formID := uuid.New()
	field := FormField{FormID: formID, DisplayName: "Email"}

	if field.TableName() != "form_fields" {
		t.Fatalf("unexpected table name")
	}
	if field.FormID != formID {
		t.Fatalf("expected form id to match")
	}
}

func TestInboundLeadFormAssociationTable(t *testing.T) {
	assoc := InboundLeadFormListContactAssociation{}
	if assoc.TableName() != "inbound_lead_form_list_contact_association" {
		t.Fatalf("unexpected association table name")
	}
}

package operation

import "testing"

func TestOperationTableName(t *testing.T) {
	if (Operation{}).TableName() != "operations" {
		t.Fatalf("unexpected table name")
	}
}

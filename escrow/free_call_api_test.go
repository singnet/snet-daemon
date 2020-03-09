package escrow

import (
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestFreeCallUserKey_String(t *testing.T) {
	key := &FreeCallUserKey{UserId: "A", OrganizationId: "B", ServiceId: "C", GroupID: "D"}
	assert.Equal(t, "{ID:A/B/C/D}", key.String())
}

func TestFreeCallUserData_String(t *testing.T) {
	data := &FreeCallUserData{FreeCallsMade: 10, UserId: "abc@test.com", OrganizationId: "org1", ServiceId: "service1", GroupID: "grp1"}
	IncrementFreeCallCount(data)
	assert.Equal(t, "{User abc@test.com has made 11 free calls for org_id=org1, service_id=service1, group_id=grp1 }", data.String())

}

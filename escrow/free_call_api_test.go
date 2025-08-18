package escrow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFreeCallUserKey_String(t *testing.T) {
	key := &FreeCallUserKey{Address: "A", UserId: "B", OrganizationId: "C", ServiceId: "D", GroupID: "F"}
	assert.Equal(t, "{ID:A/B/C/D/F}", key.String())
}

func TestFreeCallUserData_String(t *testing.T) {
	data := &FreeCallUserData{FreeCallsMade: 10, UserID: "1", Address: "0xC9D12f2EfF4B6693FA1D7E7C87deD2E1bbA589e5", OrganizationId: "org1", ServiceId: "service1", GroupID: "grp1"}
	IncrementFreeCallCount(data)
	assert.Equal(t, "{Addr:0xC9D12f2EfF4B6693FA1D7E7C87deD2E1bbA589e5 (id:1) has made 11 free calls for org_id=org1, service_id=service1, group_id=grp1 }", data.String())
}

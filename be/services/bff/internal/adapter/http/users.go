package httpadapter

import (
	"strings"

	"github.com/google/uuid"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
)

type devUser struct {
	identity tenant.Identity
	name     string
}

var devUsers = map[string]devUser{
	"an": {
		name: "An",
		identity: tenant.Identity{
			TenantID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			UserID:   "an", Capabilities: map[string]bool{"order.create": true},
		},
	},
	"binh": {
		name: "Bình",
		identity: tenant.Identity{
			TenantID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			UserID:   "binh", Capabilities: map[string]bool{"order.create": true, "order.approve": true},
		},
	},
	"chi": {
		name: "Chi",
		identity: tenant.Identity{
			TenantID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			UserID:   "chi", Capabilities: map[string]bool{"order.approve": true},
		},
	},
	"dung": {
		name: "Dũng",
		identity: tenant.Identity{
			TenantID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			UserID:   "dung", Capabilities: map[string]bool{"order.create": true, "order.approve": true},
		},
	},
	"kho": {
		name: "Kho",
		identity: tenant.Identity{
			TenantID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			UserID:   "kho", Capabilities: map[string]bool{"stock.adjust": true},
		},
	},
}

func lookupDevUser(value string) (devUser, bool) {
	user, ok := devUsers[strings.ToLower(strings.TrimSpace(value))]
	return user, ok
}

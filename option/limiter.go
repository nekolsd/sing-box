package option

import (
	"github.com/sagernet/sing/common/json/badoption"
)

type Limiter struct {
	Tag                 string                     `json:"tag"`
	Download            string                     `json:"download,omitempty"`
	Upload              string                     `json:"upload,omitempty"`
	Timeout             badoption.Duration         `json:"timeout,omitempty"`
	AuthUser            badoption.Listable[string] `json:"auth_user,omitempty"`
	AuthUserIndependent bool                       `json:"auth_user_independent,omitempty"`
	Inbound             badoption.Listable[string] `json:"inbound,omitempty"`
	InboundIndependent  bool                       `json:"inbound_independent,omitempty"`
}

package http

import (
	"testing"

	"github.com/taubyte/tau/core/services/substrate"
	"github.com/taubyte/tau/pkg/config"
)

type capped struct{ substrate.Service }

func (capped) Caps(string) config.Caps { return config.Caps{Instances: 7} }

// A function reaches the substrate through its component, which only embeds the interface.
func TestComponentReachesSubstrateCaps(t *testing.T) {
	if got := (&Service{Service: capped{}}).Caps("p").Instances; got != 7 {
		t.Fatalf("caps through the component: got %d want 7", got)
	}
}

package providers

import "testing"

func TestIntegrations_Enabled(t *testing.T) {
	SetRequestedIntegrations([]string{"Neovim"})
	if !integrationEnabled("neovim") {
		t.Fatalf("expected neovim enabled")
	}
	if integrationEnabled("emacs") {
		t.Fatalf("expected emacs disabled")
	}
}


package providers

import "testing"

func TestIntegrationReport_Consume(t *testing.T) {
	AddIntegrationReportLine("github:x/y", "v1", "Integrated into Neovim: /tmp")
	lines := ConsumeIntegrationReport("github:x/y", "v1")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %#v", lines)
	}
	again := ConsumeIntegrationReport("github:x/y", "v1")
	if len(again) != 0 {
		t.Fatalf("expected empty after consume, got %#v", again)
	}
}


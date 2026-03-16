package health

import "testing"

func TestReadyStatus(t *testing.T) {
	m := NewManager()
	m.SetReady(true)
	m.RegisterCheck("db", func() (bool, string) { return true, "connected" })
	m.RegisterCheck("skills", func() (bool, string) { return false, "missing" })

	resp, ok := m.ReadyStatus()
	if ok {
		t.Fatalf("ReadyStatus() ready = true, want false")
	}
	if resp.Status != "not ready" {
		t.Fatalf("ReadyStatus() status = %q, want %q", resp.Status, "not ready")
	}
	if resp.Checks["db"].Status != "ok" {
		t.Fatalf("db status = %q, want ok", resp.Checks["db"].Status)
	}
	if resp.Checks["skills"].Status != "fail" {
		t.Fatalf("skills status = %q, want fail", resp.Checks["skills"].Status)
	}
}

func TestHealthStatus(t *testing.T) {
	m := NewManager()
	resp := m.HealthStatus()
	if resp.Status != "ok" {
		t.Fatalf("HealthStatus() status = %q, want ok", resp.Status)
	}
	if resp.Uptime == "" {
		t.Fatal("HealthStatus() uptime is empty")
	}
}

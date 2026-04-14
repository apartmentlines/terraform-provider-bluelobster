package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientUsesConfiguredBaseURL(t *testing.T) {
	client, err := NewClient("https://api.bluelobster.ai/api/v1", "token", "test")
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}
	if got := client.baseURL.String(); got != "https://api.bluelobster.ai/api/v1" {
		t.Fatalf("unexpected base URL %q", got)
	}
}

func TestLaunchCustomInstanceIncludesGPUModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-Key"); got != "token" {
			t.Fatalf("unexpected api key header %q", got)
		}
		if r.URL.Path != "/api/v1/instances/launch-custom" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"task_id":"task-1","vm_uuid":"vm-1"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/api/v1", "token", "test")
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	_, err = client.LaunchCustomInstance(context.Background(), LaunchCustomInstanceInput{
		Name:         "custom",
		InstanceType: "gpu_custom",
		Host:         "phl-gpu-01",
		Cores:        8,
		MemoryGB:     32,
		DiskSizeGB:   100,
		GPUCount:     1,
		GPUModel:     "A4000",
		Username:     "ubuntu",
		SSHPublicKey: "ssh-ed25519 AAAA",
		TemplateName: "UBUNTU-22-04-NV",
	})
	if err != nil {
		t.Fatalf("expected launch to succeed, got error: %v", err)
	}
}

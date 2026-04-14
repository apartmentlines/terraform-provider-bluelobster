package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWaitForCreatedInstanceWaitsForTaskBeforeReadingInstance(t *testing.T) {
	var mu sync.Mutex
	taskSeen := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/tasks/task-1":
			mu.Lock()
			taskSeen = true
			mu.Unlock()
			_, _ = w.Write([]byte(`{"task_id":"task-1","status":"COMPLETED","params":{"vm_uuid":"vm-1"}}`))
		case "/api/v1/instances/vm-1":
			mu.Lock()
			defer mu.Unlock()
			if !taskSeen {
				t.Fatalf("instance was read before launch task completed")
			}
			_, _ = w.Write([]byte(`{"uuid":"vm-1","name":"example","power_status":"off","created_at":"2026-04-14T00:00:00Z"}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/api/v1", "token", "test")
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	resource := &InstanceResource{client: client}
	instance, err := resource.waitForCreatedInstance(context.Background(), LaunchResponse{
		TaskID:     "task-1",
		InstanceID: "vm-1",
	})
	if err != nil {
		t.Fatalf("expected launch wait to succeed, got error: %v", err)
	}
	if instance.ID != "vm-1" {
		t.Fatalf("unexpected instance %#v", instance)
	}
}

func TestApplyInstanceActionWaitsForTaskAndObservedState(t *testing.T) {
	var mu sync.Mutex
	taskSeen := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/instances/vm-1/power-on":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method %q", r.Method)
			}
			_, _ = w.Write([]byte(`{"task_id":"task-1","status":"accepted"}`))
		case "/api/v1/tasks/task-1":
			mu.Lock()
			taskSeen = true
			mu.Unlock()
			_, _ = w.Write([]byte(`{"task_id":"task-1","status":"COMPLETED"}`))
		case "/api/v1/instances/vm-1":
			mu.Lock()
			defer mu.Unlock()
			if !taskSeen {
				t.Fatalf("instance condition was checked before action task completed")
			}
			_, _ = w.Write([]byte(`{"uuid":"vm-1","name":"example","power_status":"running","created_at":"2026-04-14T00:00:00Z"}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/api/v1", "token", "test")
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	instance, err := applyInstanceAction(context.Background(), client, "vm-1", "power-on",
		func(ctx context.Context) (ActionResponse, error) {
			return client.PowerOnInstance(ctx, "vm-1")
		},
		func(instance VMInstance) bool {
			return normalizePowerState(instance.PowerStatus) == "running"
		},
	)
	if err != nil {
		t.Fatalf("expected action wait to succeed, got error: %v", err)
	}
	if normalizePowerState(instance.PowerStatus) != "running" {
		t.Fatalf("unexpected instance %#v", instance)
	}
}

func TestDeleteInstanceAndWaitRetriesInvalidState(t *testing.T) {
	var mu sync.Mutex
	deleteAttempts := 0
	deleted := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/instances/vm-1":
			switch r.Method {
			case http.MethodDelete:
				mu.Lock()
				deleteAttempts++
				attempt := deleteAttempts
				mu.Unlock()

				if attempt == 1 {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"detail":{"error":"invalid_state","message":"VM is not ready for deletion"}}`))
					return
				}

				_, _ = w.Write([]byte(`{"task_id":"task-del","status":"accepted"}`))
			case http.MethodGet:
				mu.Lock()
				isDeleted := deleted
				mu.Unlock()

				if isDeleted {
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"detail":{"error":"not_found","message":"missing"}}`))
					return
				}

				_, _ = w.Write([]byte(`{"uuid":"vm-1","name":"example","power_status":"off","created_at":"2026-04-14T00:00:00Z"}`))
			default:
				t.Fatalf("unexpected method %q", r.Method)
			}
		case "/api/v1/tasks/task-del":
			mu.Lock()
			deleted = true
			mu.Unlock()
			_, _ = w.Write([]byte(`{"task_id":"task-del","status":"COMPLETED"}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/api/v1", "token", "test")
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	if err := deleteInstanceAndWait(ctx, client, "vm-1"); err != nil {
		t.Fatalf("expected delete to succeed, got error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if deleteAttempts < 2 {
		t.Fatalf("expected delete retry, got %d attempts", deleteAttempts)
	}
}

func TestSyncStandardInstanceModelPreservesConfiguredRegion(t *testing.T) {
	model := &instanceResourceModel{
		Region: types.StringValue("phl"),
	}

	syncStandardInstanceModel(model, VMInstance{
		ID:           "vm-1",
		Region:       "Philadelphia",
		InstanceType: "v1_gpu_1x_2080ti",
		PowerStatus:  "running",
		CreatedAt:    "2026-04-14T00:00:00Z",
	})

	if got := model.Region.ValueString(); got != "phl" {
		t.Fatalf("expected configured region to be preserved, got %q", got)
	}
}

func TestFirewallCreateAddsRulesInReverseToPreserveConfiguredOrder(t *testing.T) {
	var postedDPorts []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/instances/vm-1/firewall":
			switch r.Method {
			case http.MethodPut:
				_, _ = w.Write([]byte(`{"status":"success"}`))
			case http.MethodGet:
				_, _ = w.Write([]byte(`{"enabled":true,"policy_in":"DROP","policy_out":"ACCEPT","rules":[{"pos":0,"action":"ACCEPT","dport":"22","proto":"tcp","enable":1,"comment":"SSH","type":"in"},{"pos":1,"action":"ACCEPT","dport":"443","proto":"tcp","enable":1,"comment":"HTTPS","type":"in"}]}`))
			default:
				t.Fatalf("unexpected method %q", r.Method)
			}
		case "/api/v1/instances/vm-1/firewall/rules":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method %q", r.Method)
			}
			raw, _ := io.ReadAll(r.Body)
			body := string(raw)
			switch {
			case strings.Contains(body, `"dport":"22"`):
				postedDPorts = append(postedDPorts, "22")
			case strings.Contains(body, `"dport":"443"`):
				postedDPorts = append(postedDPorts, "443")
			default:
				t.Fatalf("unexpected rule payload %q", body)
			}
			_, _ = w.Write([]byte(`{"status":"success"}`))
		case "/api/v1/instances/vm-1/firewall/rules/0", "/api/v1/instances/vm-1/firewall/rules/1":
			if r.Method != http.MethodDelete {
				t.Fatalf("unexpected method %q", r.Method)
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/api/v1", "token", "test")
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	firewallResource := &InstanceFirewallResource{client: client}
	plan := instanceFirewallResourceModel{
		InstanceID: types.StringValue("vm-1"),
		Enabled:    types.BoolValue(true),
		PolicyIn:   types.StringValue("DROP"),
		PolicyOut:  types.StringValue("ACCEPT"),
		Rules: buildFirewallRuleList([]FirewallRule{
			{Type: "in", Action: "ACCEPT", Proto: "tcp", DPort: "22", Comment: "SSH", Enable: 1},
			{Type: "in", Action: "ACCEPT", Proto: "tcp", DPort: "443", Comment: "HTTPS", Enable: 1},
		}),
	}

	var diags diag.Diagnostics
	firewallResource.applyFirewallPlan(context.Background(), &plan, &diags)
	if diags.HasError() {
		t.Fatalf("expected firewall apply to succeed, got %v", diags)
	}

	if strings.Join(postedDPorts, ",") != "443,22" {
		t.Fatalf("expected rules to be posted in reverse order, got %v", postedDPorts)
	}
}

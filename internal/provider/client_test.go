package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestDecodeTemplatesSupportsCLIEnvelope(t *testing.T) {
	templates, err := decodeTemplates([]byte(`{"templates":[{"name":"ubuntu","display_name":"Ubuntu","os_type":"linux"}],"total":1}`))
	if err != nil {
		t.Fatalf("expected templates to decode, got error: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(templates))
	}
	if templates[0].Name != "ubuntu" || templates[0].DisplayName != "Ubuntu" || templates[0].OSType != "linux" {
		t.Fatalf("unexpected template decoded: %#v", templates[0])
	}
}

func TestDecodeInstancesSupportsEmptyTopLevelArray(t *testing.T) {
	instances, err := decodeInstances([]byte(`[]`))
	if err != nil {
		t.Fatalf("expected instances to decode, got error: %v", err)
	}
	if len(instances) != 0 {
		t.Fatalf("expected no instances, got %d", len(instances))
	}
}

func TestListAvailableInstancesNormalizesGPUModelArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/instances/available" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpu.large","instance_type":{"name":"gpu.large","description":"GPU large","gpu_description":"","price_cents_per_hour":1000,"specs":{"vcpus":8,"memory_gib":32,"storage_gib":200,"gpus":2,"gpu_model":["RTX 4090","RTX 6000 Ada"]}},"regions_with_capacity_available":[]}]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/api/v1", "token", "test")
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	items, err := client.ListAvailableInstances(context.Background())
	if err != nil {
		t.Fatalf("expected available instances to decode, got error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if got := items[0].InstanceType.Specs.GPUModel; got != "RTX 4090, RTX 6000 Ada" {
		t.Fatalf("unexpected gpu_model %q", got)
	}
}

func TestDecodeLaunchResponseSupportsCLIShapes(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantTaskID string
		wantID     string
		wantIP     string
	}{
		{
			name:       "nested task and vm uuid list",
			body:       `{"data":{"vm_uuids":["vm-1"],"ip_address":"198.51.100.10","task":{"id":"task-1"}}}`,
			wantTaskID: "task-1",
			wantID:     "vm-1",
			wantIP:     "198.51.100.10",
		},
		{
			name:       "top level fields",
			body:       `{"task_id":"task-2","instance_id":"vm-2","assigned_ip":"198.51.100.11"}`,
			wantTaskID: "task-2",
			wantID:     "vm-2",
			wantIP:     "198.51.100.11",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := decodeLaunchResponse([]byte(tc.body))
			if err != nil {
				t.Fatalf("expected launch response to decode, got error: %v", err)
			}
			if resp.TaskID != tc.wantTaskID || resp.InstanceID != tc.wantID || resp.AssignedIP != tc.wantIP {
				t.Fatalf("unexpected launch response: %#v", resp)
			}
		})
	}
}

func TestDecodeIPListSupportsWrappedObjectShapes(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantIPs []string
	}{
		{
			name:    "wrapped ips objects",
			body:    `{"ips":[{"ip_address":"198.51.100.20"},{"ip_address":"198.51.100.21"}]}`,
			wantIPs: []string{"198.51.100.20", "198.51.100.21"},
		},
		{
			name:    "wrapped data objects",
			body:    `{"data":[{"ip_address":"198.51.100.22"}]}`,
			wantIPs: []string{"198.51.100.22"},
		},
		{
			name:    "wrapped items objects",
			body:    `{"items":[{"ip_address":"198.51.100.23"}]}`,
			wantIPs: []string{"198.51.100.23"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ips, err := decodeIPList([]byte(tc.body))
			if err != nil {
				t.Fatalf("expected IP list to decode, got error: %v", err)
			}
			if strings.Join(ips, ",") != strings.Join(tc.wantIPs, ",") {
				t.Fatalf("unexpected IPs %v", ips)
			}
		})
	}
}

func TestReleaseInstanceIPEscapesPathSegment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method %q", r.Method)
		}
		if r.URL.Path != "/api/v1/instances/vm-1/ips/198.51.100.10%2F32" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/api/v1", "token", "test")
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	if err := client.ReleaseInstanceIP(context.Background(), "vm-1", "198.51.100.10/32"); err != nil {
		t.Fatalf("expected release to succeed, got error: %v", err)
	}
}

func TestWaitForTaskTreatsCancelledAsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tasks/task-1" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"task_id":"task-1","status":"CANCELLED","message":"cancelled by operator"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL+"/api/v1", "token", "test")
	if err != nil {
		t.Fatalf("expected client, got error: %v", err)
	}

	if _, err := client.WaitForTask(context.Background(), "task-1"); err == nil || !strings.Contains(err.Error(), "cancelled by operator") {
		t.Fatalf("expected cancelled task to fail, got %v", err)
	}
}

func TestDecodeBackupScheduleSupportsScheduleEnvelope(t *testing.T) {
	schedule, err := decodeBackupSchedule([]byte(`{"schedule":{"id":"sched-1","enabled":true,"frequency":"daily","hour_utc":3,"day_of_week":null,"day_of_month":null}}`))
	if err != nil {
		t.Fatalf("expected backup schedule to decode, got error: %v", err)
	}
	if schedule.Frequency != "daily" || schedule.HourUTC != 3 {
		t.Fatalf("unexpected schedule %#v", schedule)
	}
}

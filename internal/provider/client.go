package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var ErrNotFound = errors.New("resource not found")

const (
	defaultHTTPTimeout = 120 * time.Second
	taskPollInterval   = 3 * time.Second
)

type Client struct {
	baseURL    *url.URL
	apiKey     string
	userAgent  string
	httpClient *http.Client
}

func NewClient(rawBaseURL, apiKey, version string) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawBaseURL))
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/")

	userAgent := "terraform-provider-bluelobster"
	if strings.TrimSpace(version) != "" {
		userAgent += "/" + strings.TrimSpace(version)
	}

	return &Client{
		baseURL:   parsed,
		apiKey:    apiKey,
		userAgent: userAgent,
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
	}, nil
}

type apiError struct {
	StatusCode int
	Body       string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("Blue Lobster API returned HTTP %d: %s", e.StatusCode, e.Body)
}

func (e *apiError) code() string {
	var detail struct {
		Detail struct {
			Error string `json:"error"`
		} `json:"detail"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(e.Body), &detail); err != nil {
		return ""
	}
	return firstNonEmptyString(detail.Detail.Error, detail.Error)
}

func isAPIInvalidState(err error) bool {
	var apiErr *apiError
	return errors.As(err, &apiErr) && apiErr.code() == "invalid_state"
}

type Task struct {
	ID        string         `json:"task_id"`
	Status    string         `json:"status"`
	Operation string         `json:"operation"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
	Message   string         `json:"message"`
	AccountID string         `json:"account_id"`
	Params    map[string]any `json:"params"`
}

type VMInstance struct {
	ID                  string            `json:"uuid" tfsdk:"id"`
	Name                string            `json:"name" tfsdk:"name"`
	HostID              string            `json:"host_id" tfsdk:"host_id"`
	Region              string            `json:"region" tfsdk:"region"`
	IPAddress           string            `json:"ip_address" tfsdk:"ip_address"`
	InternalIP          string            `json:"internal_ip" tfsdk:"internal_ip"`
	CPUCores            int64             `json:"cpu_cores" tfsdk:"cpu_cores"`
	MemoryGB            int64             `json:"memory" tfsdk:"memory"`
	StorageGB           int64             `json:"storage" tfsdk:"storage"`
	GPUCount            int64             `json:"gpu_count" tfsdk:"gpu_count"`
	GPUModel            string            `json:"gpu_model" tfsdk:"gpu_model"`
	PowerStatus         string            `json:"power_status" tfsdk:"power_status"`
	CreatedAt           string            `json:"created_at" tfsdk:"created_at"`
	Metadata            map[string]string `json:"metadata" tfsdk:"metadata"`
	InstanceType        string            `json:"instance_type" tfsdk:"instance_type"`
	PriceCentsPerHour   *int64            `json:"price_cents_per_hour" tfsdk:"price_cents_per_hour"`
	TeamID              string            `json:"team_id" tfsdk:"team_id"`
	TeamName            string            `json:"team_name" tfsdk:"team_name"`
	AccessType          string            `json:"access_type" tfsdk:"access_type"`
	TemplateName        string            `json:"template_name" tfsdk:"template_name"`
	TemplateDisplayName string            `json:"template_display_name" tfsdk:"template_display_name"`
	OSType              string            `json:"os_type" tfsdk:"os_type"`
	VMUsername          string            `json:"vm_username" tfsdk:"vm_username"`
}

type LaunchStandardInstanceInput struct {
	Region       string
	InstanceType string
	Name         string
	Username     string
	SSHPublicKey string
	Password     string
	Metadata     map[string]string
	TemplateName string
	ISOURL       string
}

type LaunchResponse struct {
	TaskID     string
	InstanceID string
	AssignedIP string
}

type ActionResponse struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	TaskID     string `json:"task_id"`
	InstanceID string `json:"instance_id"`
	Host       string `json:"host"`
	DurationMS int64  `json:"duration_ms"`
	NewName    string `json:"new_name"`
}

type AvailableInstanceType struct {
	ID                           string                    `json:"id" tfsdk:"id"`
	InstanceType                 AvailableInstanceConfig   `json:"instance_type" tfsdk:"instance_type"`
	RegionsWithCapacityAvailable []AvailableInstanceRegion `json:"regions_with_capacity_available" tfsdk:"regions_with_capacity_available"`
}

type AvailableInstanceConfig struct {
	Name              string                 `json:"name" tfsdk:"name"`
	Description       string                 `json:"description" tfsdk:"description"`
	GPUDescription    string                 `json:"gpu_description" tfsdk:"gpu_description"`
	PriceCentsPerHour int64                  `json:"price_cents_per_hour" tfsdk:"price_cents_per_hour"`
	Specs             AvailableInstanceSpecs `json:"specs" tfsdk:"specs"`
}

type AvailableInstanceSpecs struct {
	VCPUs      int64  `json:"vcpus" tfsdk:"vcpus"`
	MemoryGiB  int64  `json:"memory_gib" tfsdk:"memory_gib"`
	StorageGiB int64  `json:"storage_gib" tfsdk:"storage_gib"`
	GPUs       int64  `json:"gpus" tfsdk:"gpus"`
	GPUModel   string `json:"gpu_model" tfsdk:"gpu_model"`
}

func (s *AvailableInstanceSpecs) UnmarshalJSON(data []byte) error {
	type rawSpecs struct {
		VCPUs      int64           `json:"vcpus"`
		MemoryGiB  int64           `json:"memory_gib"`
		StorageGiB int64           `json:"storage_gib"`
		GPUs       int64           `json:"gpus"`
		GPUModel   json.RawMessage `json:"gpu_model"`
	}

	var raw rawSpecs
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	s.VCPUs = raw.VCPUs
	s.MemoryGiB = raw.MemoryGiB
	s.StorageGiB = raw.StorageGiB
	s.GPUs = raw.GPUs

	if len(raw.GPUModel) == 0 || string(raw.GPUModel) == "null" {
		s.GPUModel = ""
		return nil
	}

	var single string
	if err := json.Unmarshal(raw.GPUModel, &single); err == nil {
		s.GPUModel = strings.TrimSpace(single)
		return nil
	}

	var list []string
	if err := json.Unmarshal(raw.GPUModel, &list); err == nil {
		items := make([]string, 0, len(list))
		for _, item := range list {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				items = append(items, trimmed)
			}
		}
		s.GPUModel = strings.Join(items, ", ")
		return nil
	}

	return fmt.Errorf("decode available instance specs response: unsupported gpu_model shape")
}

type AvailableInstanceRegion struct {
	Name        string                    `json:"name" tfsdk:"name"`
	Description string                    `json:"description" tfsdk:"description"`
	Location    AvailableInstanceLocation `json:"location" tfsdk:"location"`
}

type AvailableInstanceLocation struct {
	City    string `json:"city" tfsdk:"city"`
	State   string `json:"state" tfsdk:"state"`
	Country string `json:"country" tfsdk:"country"`
}

type Template struct {
	Name        string `json:"name" tfsdk:"name"`
	DisplayName string `json:"display_name" tfsdk:"display_name"`
	OSType      string `json:"os_type" tfsdk:"os_type"`
}

type FirewallOptions struct {
	Enabled   bool
	PolicyIn  string
	PolicyOut string
}

type FirewallRule struct {
	Pos     int64  `json:"pos"`
	Type    string `json:"type"`
	Action  string `json:"action"`
	Source  string `json:"source"`
	Dest    string `json:"dest"`
	Proto   string `json:"proto"`
	DPort   string `json:"dport"`
	SPort   string `json:"sport"`
	Comment string `json:"comment"`
	Enable  int64  `json:"enable"`
}

type FirewallStatus struct {
	Enabled   bool           `json:"enabled"`
	PolicyIn  string         `json:"policy_in"`
	PolicyOut string         `json:"policy_out"`
	Rules     []FirewallRule `json:"rules"`
}

type BackupSchedule struct {
	Frequency  string `json:"frequency"`
	HourUTC    int64  `json:"hour_utc"`
	DayOfWeek  *int64 `json:"day_of_week"`
	DayOfMonth *int64 `json:"day_of_month"`
}

type Backup struct {
	VolID     string `json:"volid" tfsdk:"volid"`
	SizeBytes int64  `json:"size_bytes" tfsdk:"size_bytes"`
	SizeHuman string `json:"size_human" tfsdk:"size_human"`
	CreatedAt string `json:"created_at" tfsdk:"created_at"`
	Format    string `json:"format" tfsdk:"format"`
	Notes     string `json:"notes" tfsdk:"notes"`
}

type InstanceBackups struct {
	InstanceID string   `json:"instance_id"`
	Storage    string   `json:"storage"`
	Backups    []Backup `json:"backups"`
	Total      int64    `json:"total"`
}

func (c *Client) ListAvailableInstances(ctx context.Context) ([]AvailableInstanceType, error) {
	body, err := c.doJSON(ctx, http.MethodGet, "/instances/available", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data []AvailableInstanceType `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode available instances response: %w", err)
	}
	return resp.Data, nil
}

func (c *Client) ListTemplates(ctx context.Context) ([]Template, error) {
	body, err := c.doJSON(ctx, http.MethodGet, "/instances/templates", nil)
	if err != nil {
		return nil, err
	}

	templates, err := decodeTemplates(body)
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func decodeTemplates(body []byte) ([]Template, error) {
	var list []Template
	if err := json.Unmarshal(body, &list); err == nil && len(list) > 0 {
		return list, nil
	}

	var wrapped struct {
		Data      []Template `json:"data"`
		Templates []Template `json:"templates"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil {
		switch {
		case len(wrapped.Data) > 0:
			return wrapped.Data, nil
		case len(wrapped.Templates) > 0:
			return wrapped.Templates, nil
		}
	}

	var names []string
	if err := json.Unmarshal(body, &names); err == nil && len(names) > 0 {
		templates := make([]Template, 0, len(names))
		for _, name := range names {
			templates = append(templates, Template{Name: name, DisplayName: name})
		}
		return templates, nil
	}

	return nil, fmt.Errorf("decode templates response: unsupported response shape")
}

func (c *Client) ListInstances(ctx context.Context) ([]VMInstance, error) {
	body, err := c.doJSON(ctx, http.MethodGet, "/instances", nil)
	if err != nil {
		return nil, err
	}
	return decodeInstances(body)
}

func decodeInstances(body []byte) ([]VMInstance, error) {
	var list []VMInstance
	if err := json.Unmarshal(body, &list); err == nil {
		return list, nil
	}

	var wrapped struct {
		Data      []VMInstance `json:"data"`
		Instances []VMInstance `json:"instances"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("decode instances response: %w", err)
	}
	switch {
	case len(wrapped.Data) > 0:
		return wrapped.Data, nil
	case len(wrapped.Instances) > 0:
		return wrapped.Instances, nil
	default:
		return []VMInstance{}, nil
	}
}

func (c *Client) GetInstance(ctx context.Context, id string) (VMInstance, error) {
	body, err := c.doJSON(ctx, http.MethodGet, "/instances/"+id, nil)
	if err != nil {
		return VMInstance{}, err
	}

	var instance VMInstance
	if err := json.Unmarshal(body, &instance); err != nil {
		return VMInstance{}, fmt.Errorf("decode instance response: %w", err)
	}
	if instance.ID == "" {
		return VMInstance{}, fmt.Errorf("decode instance response: missing uuid")
	}
	return instance, nil
}

func (c *Client) LaunchStandardInstance(ctx context.Context, input LaunchStandardInstanceInput) (LaunchResponse, error) {
	payload := map[string]any{
		"region":        input.Region,
		"instance_type": input.InstanceType,
		"username":      input.Username,
	}
	if input.Name != "" {
		payload["name"] = input.Name
	}
	if input.SSHPublicKey != "" {
		payload["ssh_key"] = input.SSHPublicKey
	}
	if input.Password != "" {
		payload["password"] = input.Password
	}
	if len(input.Metadata) > 0 {
		payload["metadata"] = input.Metadata
	}
	if input.TemplateName != "" {
		payload["template_name"] = input.TemplateName
	}
	if input.ISOURL != "" {
		payload["iso_url"] = input.ISOURL
	}

	body, err := c.doJSON(ctx, http.MethodPost, "/instances/launch-instance", payload)
	if err != nil {
		return LaunchResponse{}, err
	}
	return decodeLaunchResponse(body)
}

func decodeLaunchResponse(body []byte) (LaunchResponse, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return LaunchResponse{}, fmt.Errorf("decode launch response: %w", err)
	}

	data, _ := raw["data"].(map[string]any)
	task, _ := raw["task"].(map[string]any)
	dataTask, _ := data["task"].(map[string]any)

	instanceID := firstNonEmptyString(
		firstStringFromList(raw["instance_ids"]),
		firstStringFromList(raw["vm_uuids"]),
		stringValue(raw["vm_uuid"]),
		stringValue(raw["instance_id"]),
		firstStringFromList(data["instance_ids"]),
		firstStringFromList(data["vm_uuids"]),
		stringValue(data["vm_uuid"]),
		stringValue(data["instance_id"]),
	)

	taskID := firstNonEmptyString(
		stringValue(raw["task_id"]),
		stringValue(task["task_id"]),
		stringValue(task["id"]),
		stringValue(data["task_id"]),
		stringValue(dataTask["task_id"]),
		stringValue(dataTask["id"]),
	)
	if taskID == "" && instanceID == "" {
		return LaunchResponse{}, fmt.Errorf("decode launch response: missing task_id and instance identifier")
	}

	return LaunchResponse{
		TaskID:     taskID,
		InstanceID: instanceID,
		AssignedIP: firstNonEmptyString(
			stringValue(raw["assigned_ip"]),
			stringValue(raw["ip_address"]),
			stringValue(data["assigned_ip"]),
			stringValue(data["ip_address"]),
		),
	}, nil
}

func (c *Client) RenameInstance(ctx context.Context, id, name string) (ActionResponse, error) {
	body, err := c.doJSON(ctx, http.MethodPut, "/instances/"+id+"/rename", map[string]any{"name": name})
	if err != nil {
		return ActionResponse{}, err
	}
	return decodeActionResponse(body)
}

func (c *Client) PowerOnInstance(ctx context.Context, id string) (ActionResponse, error) {
	body, err := c.doJSON(ctx, http.MethodPost, "/instances/"+id+"/power-on", nil)
	if err != nil {
		return ActionResponse{}, err
	}
	return decodeActionResponse(body)
}

func (c *Client) ShutdownInstance(ctx context.Context, id string) (ActionResponse, error) {
	body, err := c.doJSON(ctx, http.MethodPost, "/instances/"+id+"/shutdown", nil)
	if err != nil {
		return ActionResponse{}, err
	}
	return decodeActionResponse(body)
}

func (c *Client) DeleteInstance(ctx context.Context, id string) (ActionResponse, error) {
	body, err := c.doJSON(ctx, http.MethodDelete, "/instances/"+id, nil)
	if errors.Is(err, ErrNotFound) {
		return ActionResponse{}, nil
	}
	if err != nil {
		return ActionResponse{}, err
	}
	return decodeActionResponse(body)
}

func decodeActionResponse(body []byte) (ActionResponse, error) {
	if len(body) == 0 {
		return ActionResponse{}, nil
	}

	var response ActionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return ActionResponse{}, fmt.Errorf("decode action response: %w", err)
	}
	return response, nil
}

func (c *Client) GetTask(ctx context.Context, id string) (Task, error) {
	body, err := c.doJSON(ctx, http.MethodGet, "/tasks/"+id, nil)
	if err != nil {
		return Task{}, err
	}
	var task Task
	if err := json.Unmarshal(body, &task); err != nil {
		return Task{}, fmt.Errorf("decode task response: %w", err)
	}
	if task.ID == "" {
		return Task{}, fmt.Errorf("decode task response: missing task_id")
	}
	return task, nil
}

func (c *Client) WaitForTask(ctx context.Context, id string) (Task, error) {
	ticker := time.NewTicker(taskPollInterval)
	defer ticker.Stop()

	for {
		task, err := c.GetTask(ctx, id)
		if err != nil {
			return Task{}, err
		}

		switch strings.ToUpper(strings.TrimSpace(task.Status)) {
		case "COMPLETED":
			return task, nil
		case "FAILED", "CANCELLED":
			if task.Message == "" {
				task.Message = "task failed"
			}
			return Task{}, fmt.Errorf("task %s failed: %s", id, task.Message)
		}

		select {
		case <-ctx.Done():
			return Task{}, fmt.Errorf("wait for task %s: %w", id, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (c *Client) WaitForInstanceVisible(ctx context.Context, id string) (VMInstance, error) {
	ticker := time.NewTicker(taskPollInterval)
	defer ticker.Stop()

	for {
		instance, err := c.GetInstance(ctx, id)
		if err == nil {
			return instance, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return VMInstance{}, err
		}
		select {
		case <-ctx.Done():
			return VMInstance{}, fmt.Errorf("wait for instance %s: %w", id, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (c *Client) WaitForPowerState(ctx context.Context, id string, targets ...string) (VMInstance, error) {
	want := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		want[normalizePowerState(target)] = struct{}{}
	}

	ticker := time.NewTicker(taskPollInterval)
	defer ticker.Stop()

	for {
		instance, err := c.GetInstance(ctx, id)
		if err != nil {
			return VMInstance{}, err
		}
		if _, ok := want[normalizePowerState(instance.PowerStatus)]; ok {
			return instance, nil
		}
		select {
		case <-ctx.Done():
			return VMInstance{}, fmt.Errorf("wait for instance %s power state %v: %w", id, targets, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (c *Client) WaitForDeletion(ctx context.Context, id string) error {
	ticker := time.NewTicker(taskPollInterval)
	defer ticker.Stop()

	for {
		_, err := c.GetInstance(ctx, id)
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for instance %s deletion: %w", id, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (c *Client) GetFirewall(ctx context.Context, instanceID string) (FirewallStatus, error) {
	body, err := c.doJSON(ctx, http.MethodGet, "/instances/"+instanceID+"/firewall", nil)
	if err != nil {
		return FirewallStatus{}, err
	}
	var status FirewallStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return FirewallStatus{}, fmt.Errorf("decode firewall response: %w", err)
	}
	return status, nil
}

func (c *Client) UpdateFirewall(ctx context.Context, instanceID string, options FirewallOptions) error {
	payload := map[string]any{
		"enable":     options.Enabled,
		"policy_in":  options.PolicyIn,
		"policy_out": options.PolicyOut,
	}
	_, err := c.doJSON(ctx, http.MethodPut, "/instances/"+instanceID+"/firewall", payload)
	return err
}

func (c *Client) AddFirewallRule(ctx context.Context, instanceID string, rule FirewallRule) error {
	payload := map[string]any{
		"type":   rule.Type,
		"action": rule.Action,
	}
	if rule.Source != "" {
		payload["source"] = rule.Source
	}
	if rule.Dest != "" {
		payload["dest"] = rule.Dest
	}
	if rule.Proto != "" {
		payload["proto"] = rule.Proto
	}
	if rule.DPort != "" {
		payload["dport"] = rule.DPort
	}
	if rule.SPort != "" {
		payload["sport"] = rule.SPort
	}
	if rule.Comment != "" {
		payload["comment"] = rule.Comment
	}
	payload["enable"] = rule.Enable
	_, err := c.doJSON(ctx, http.MethodPost, "/instances/"+instanceID+"/firewall/rules", payload)
	return err
}

func (c *Client) DeleteFirewallRule(ctx context.Context, instanceID string, pos int64) error {
	_, err := c.doJSON(ctx, http.MethodDelete, fmt.Sprintf("/instances/%s/firewall/rules/%d", instanceID, pos), nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func (c *Client) GetBackupSchedule(ctx context.Context, instanceID string) (BackupSchedule, error) {
	body, err := c.doJSON(ctx, http.MethodGet, "/instances/"+instanceID+"/backup-schedule", nil)
	if err != nil {
		return BackupSchedule{}, err
	}
	return decodeBackupSchedule(body)
}

func decodeBackupSchedule(body []byte) (BackupSchedule, error) {
	var schedule BackupSchedule
	if err := json.Unmarshal(body, &schedule); err == nil && schedule.Frequency != "" {
		return schedule, nil
	}
	var wrapped struct {
		Data     BackupSchedule `json:"data"`
		Schedule BackupSchedule `json:"schedule"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return BackupSchedule{}, fmt.Errorf("decode backup schedule response: %w", err)
	}
	switch {
	case wrapped.Data.Frequency != "":
		return wrapped.Data, nil
	case wrapped.Schedule.Frequency != "":
		return wrapped.Schedule, nil
	default:
		return BackupSchedule{}, fmt.Errorf("decode backup schedule response: missing frequency")
	}
}

func (c *Client) UpsertBackupSchedule(ctx context.Context, instanceID string, schedule BackupSchedule) error {
	payload := map[string]any{
		"frequency": schedule.Frequency,
		"hour_utc":  schedule.HourUTC,
	}
	if schedule.DayOfWeek != nil {
		payload["day_of_week"] = *schedule.DayOfWeek
	}
	if schedule.DayOfMonth != nil {
		payload["day_of_month"] = *schedule.DayOfMonth
	}
	_, err := c.doJSON(ctx, http.MethodPost, "/instances/"+instanceID+"/backup-schedule", payload)
	return err
}

func (c *Client) DeleteBackupSchedule(ctx context.Context, instanceID string) error {
	_, err := c.doJSON(ctx, http.MethodDelete, "/instances/"+instanceID+"/backup-schedule", nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func (c *Client) ListInstanceIPs(ctx context.Context, instanceID string) ([]string, error) {
	body, err := c.doJSON(ctx, http.MethodGet, "/instances/"+instanceID+"/ips", nil)
	if err != nil {
		return nil, err
	}
	return decodeIPList(body)
}

func decodeIPList(body []byte) ([]string, error) {
	var list []string
	if err := json.Unmarshal(body, &list); err == nil {
		return list, nil
	}

	var wrapped struct {
		IPs   []any `json:"ips"`
		Data  []any `json:"data"`
		Items []any `json:"items"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil {
		switch {
		case wrapped.IPs != nil:
			return decodeIPEntries(wrapped.IPs)
		case wrapped.Data != nil:
			return decodeIPEntries(wrapped.Data)
		case wrapped.Items != nil:
			return decodeIPEntries(wrapped.Items)
		}
	}

	var generic []map[string]any
	if err := json.Unmarshal(body, &generic); err == nil {
		entries := make([]any, 0, len(generic))
		for _, item := range generic {
			entries = append(entries, item)
		}
		return decodeIPEntries(entries)
	}

	return nil, fmt.Errorf("decode IP list response: unsupported response shape")
}

func decodeIPEntries(entries []any) ([]string, error) {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		switch value := entry.(type) {
		case string:
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				out = append(out, trimmed)
			}
		case map[string]any:
			ip := firstNonEmptyString(stringValue(value["ip_address"]), stringValue(value["address"]))
			if ip != "" {
				out = append(out, ip)
			}
		}
	}
	return out, nil
}

func (c *Client) AssignInstanceIP(ctx context.Context, instanceID string) error {
	_, err := c.doJSON(ctx, http.MethodPost, "/instances/"+instanceID+"/ips", nil)
	return err
}

func (c *Client) ReleaseInstanceIP(ctx context.Context, instanceID, ip string) error {
	_, err := c.doJSON(ctx, http.MethodDelete, "/instances/"+instanceID+"/ips/"+url.PathEscape(ip), nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func (c *Client) WaitForNewIP(ctx context.Context, instanceID string, previous []string) (string, error) {
	known := make(map[string]struct{}, len(previous))
	for _, ip := range previous {
		known[ip] = struct{}{}
	}

	ticker := time.NewTicker(taskPollInterval)
	defer ticker.Stop()

	for {
		current, err := c.ListInstanceIPs(ctx, instanceID)
		if err != nil {
			return "", err
		}
		sort.Strings(current)
		for _, ip := range current {
			if _, ok := known[ip]; !ok {
				return ip, nil
			}
		}
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("wait for new instance IP on %s: %w", instanceID, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (c *Client) ListInstanceBackups(ctx context.Context, instanceID string) (InstanceBackups, error) {
	body, err := c.doJSON(ctx, http.MethodGet, "/instances/"+instanceID+"/backups", nil)
	if err != nil {
		return InstanceBackups{}, err
	}
	var backups InstanceBackups
	if err := json.Unmarshal(body, &backups); err != nil {
		return InstanceBackups{}, fmt.Errorf("decode instance backups response: %w", err)
	}
	return backups, nil
}

func (c *Client) doJSON(ctx context.Context, method, endpoint string, payload any) ([]byte, error) {
	endpointURL := *c.baseURL
	endpointURL.Path = path.Join(endpointURL.Path, strings.TrimPrefix(endpoint, "/"))

	var requestBody io.Reader
	var requestBytes []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		requestBytes = encoded
		requestBody = bytes.NewReader(encoded)
	}

	requestURL := endpointURL.String()
	tflog.Debug(ctx, "bluelobster api request", map[string]any{
		"method":      method,
		"url":         requestURL,
		"has_payload": payload != nil,
	})

	req, err := http.NewRequestWithContext(ctx, method, requestURL, requestBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-API-Key", c.apiKey)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	tflog.Trace(ctx, "bluelobster api response", map[string]any{
		"method":   method,
		"url":      requestURL,
		"status":   resp.StatusCode,
		"response": string(body),
		"request":  string(requestBytes),
	})

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &apiError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}
	return body, nil
}

func normalizePowerState(state string) string {
	return strings.ToLower(strings.TrimSpace(state))
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstStringFromList(v any) string {
	items, ok := v.([]any)
	if !ok {
		return ""
	}
	for _, item := range items {
		if value := stringValue(item); value != "" {
			return value
		}
	}
	return ""
}

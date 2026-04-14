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

type LaunchCustomInstanceInput struct {
	Name         string
	InstanceType string
	Host         string
	Cores        int64
	MemoryGB     int64
	DiskSizeGB   int64
	GPUCount     int64
	GPUModel     string
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
		Data []Template `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && len(wrapped.Data) > 0 {
		return wrapped.Data, nil
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
	if err := json.Unmarshal(body, &list); err == nil && len(list) > 0 {
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

func (c *Client) LaunchCustomInstance(ctx context.Context, input LaunchCustomInstanceInput) (LaunchResponse, error) {
	payload := map[string]any{
		"name":          input.Name,
		"instance_type": input.InstanceType,
		"host":          input.Host,
		"cores":         input.Cores,
		"memory":        input.MemoryGB,
		"disk_size":     input.DiskSizeGB,
		"gpu_count":     input.GPUCount,
		"gpu_model":     input.GPUModel,
		"username":      input.Username,
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

	body, err := c.doJSON(ctx, http.MethodPost, "/instances/launch-custom", payload)
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
	instanceID := firstNonEmptyString(
		stringValue(raw["vm_uuid"]),
		stringValue(raw["instance_id"]),
		stringValue(data["vm_uuid"]),
		stringValue(data["instance_id"]),
	)
	if instanceID == "" {
		if ids, ok := data["instance_ids"].([]any); ok && len(ids) > 0 {
			instanceID = stringValue(ids[0])
		}
	}

	taskID := firstNonEmptyString(stringValue(raw["task_id"]), stringValue(data["task_id"]))
	if taskID == "" && instanceID == "" {
		return LaunchResponse{}, fmt.Errorf("decode launch response: missing task_id and instance identifier")
	}

	return LaunchResponse{
		TaskID:     taskID,
		InstanceID: instanceID,
		AssignedIP: firstNonEmptyString(stringValue(raw["assigned_ip"]), stringValue(data["assigned_ip"])),
	}, nil
}

func (c *Client) RenameInstance(ctx context.Context, id, name string) error {
	_, err := c.doJSON(ctx, http.MethodPut, "/instances/"+id+"/rename", map[string]any{"name": name})
	return err
}

func (c *Client) PowerOnInstance(ctx context.Context, id string) error {
	_, err := c.doJSON(ctx, http.MethodPost, "/instances/"+id+"/power-on", nil)
	return err
}

func (c *Client) ShutdownInstance(ctx context.Context, id string) error {
	_, err := c.doJSON(ctx, http.MethodPost, "/instances/"+id+"/shutdown", nil)
	return err
}

func (c *Client) DeleteInstance(ctx context.Context, id string) error {
	_, err := c.doJSON(ctx, http.MethodDelete, "/instances/"+id, nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
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
		case "FAILED":
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
		Data BackupSchedule `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return BackupSchedule{}, fmt.Errorf("decode backup schedule response: %w", err)
	}
	if wrapped.Data.Frequency == "" {
		return BackupSchedule{}, fmt.Errorf("decode backup schedule response: missing frequency")
	}
	return wrapped.Data, nil
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
		IPs []string `json:"ips"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.IPs != nil {
		return wrapped.IPs, nil
	}

	var generic []map[string]any
	if err := json.Unmarshal(body, &generic); err == nil {
		out := make([]string, 0, len(generic))
		for _, item := range generic {
			ip := firstNonEmptyString(stringValue(item["ip_address"]), stringValue(item["address"]))
			if ip != "" {
				out = append(out, ip)
			}
		}
		return out, nil
	}

	return nil, fmt.Errorf("decode IP list response: unsupported response shape")
}

func (c *Client) AssignInstanceIP(ctx context.Context, instanceID string) error {
	_, err := c.doJSON(ctx, http.MethodPost, "/instances/"+instanceID+"/ips", nil)
	return err
}

func (c *Client) ReleaseInstanceIP(ctx context.Context, instanceID, ip string) error {
	_, err := c.doJSON(ctx, http.MethodDelete, "/instances/"+instanceID+"/ips/"+ip, nil)
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

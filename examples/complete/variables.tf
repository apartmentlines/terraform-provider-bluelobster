variable "create_standard_instance" {
  description = "Create a standard instance and the dependent resources/data sources that hang off it."
  type        = bool
  default     = false
}

variable "manage_supporting_resources" {
  description = "Manage firewall, backup schedule, and extra IP for the standard instance when it exists."
  type        = bool
  default     = true
}

variable "standard_instance_name" {
  description = "Name for the standard instance."
  type        = string
  default     = "tf-standard-test"
}

variable "standard_region" {
  description = "Region for the standard instance."
  type        = string
  default     = "igl"
}

variable "standard_instance_type" {
  description = "Instance type for the standard instance."
  type        = string
  default     = "v1_gpu_1x_a5000"
}

variable "standard_username" {
  description = "OS username for the standard instance."
  type        = string
  default     = "ubuntu"
}

variable "standard_template_name" {
  description = "Template name for the standard instance."
  type        = string
  default     = "UBUNTU-22-04-NV"
}

variable "standard_ssh_public_key_path" {
  description = "Path to the SSH public key to inject into the standard instance."
  type        = string
  default     = "~/.ssh/id_rsa.pub"
}

variable "standard_power_state" {
  description = "Desired power state for the standard instance."
  type        = string
  default     = "running"
}

variable "backup_frequency" {
  description = "Backup schedule frequency for the standard instance."
  type        = string
  default     = "daily"
}

variable "backup_hour_utc" {
  description = "Backup hour in UTC for the standard instance."
  type        = number
  default     = 3
}

variable "backup_day_of_week" {
  description = "Day of week for weekly backup schedules."
  type        = number
  default     = null
  nullable    = true
}

variable "backup_day_of_month" {
  description = "Day of month for monthly backup schedules."
  type        = number
  default     = null
  nullable    = true
}

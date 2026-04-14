resource "bluelobster_custom_instance" "custom" {
  name            = "custom-gpu-instance"
  instance_type   = "gpu_custom"
  host            = "phl-gpu-01"
  cores           = 16
  memory_size     = 64
  disk_size       = 200
  gpu_count_input = 2
  gpu_model_input = "A4000"
  username        = "ubuntu"

  template_name     = "UBUNTU-22-04-NV"
  ssh_public_key_wo = file("~/.ssh/id_ed25519.pub")
}

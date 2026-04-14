resource "bluelobster_instance" "worker" {
  region        = "igl"
  instance_type = "v1_gpu_1x_a5000"
  username      = "ubuntu"
  name          = "worker-1"

  template_name     = "UBUNTU-22-04-NV"
  ssh_public_key_wo = file(pathexpand("~/.ssh/id_ed25519.pub"))
  power_state       = "running"
}

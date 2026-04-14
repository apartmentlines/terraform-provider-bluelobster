resource "bluelobster_instance_firewall" "worker" {
  instance_id = bluelobster_instance.worker.id
  enabled     = true
  policy_in   = "DROP"
  policy_out  = "ACCEPT"

  rules = [
    {
      type    = "in"
      action  = "ACCEPT"
      proto   = "tcp"
      dport   = "22"
      comment = "SSH"
      enabled = true
    },
    {
      type    = "in"
      action  = "ACCEPT"
      proto   = "tcp"
      dport   = "80,443"
      comment = "Web"
      enabled = true
    }
  ]
}

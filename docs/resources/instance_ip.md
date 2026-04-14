# bluelobster_instance_ip

Allocates one additional IP address to a Blue Lobster instance and tracks the assigned address in state.

## Argument Reference

- `instance_id` (Required, Forces replacement) Instance ID to which the additional IP will be assigned.

## Attribute Reference

- `id` Resource ID in the form `<instance_id>,<ip_address>`.
- `ip_address` The allocated IP address attached to the instance.

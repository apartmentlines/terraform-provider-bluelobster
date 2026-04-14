data "bluelobster_templates" "all" {}

output "template_names" {
  value = [for template in data.bluelobster_templates.all.templates : template.name]
}

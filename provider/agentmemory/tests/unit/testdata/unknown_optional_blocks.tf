variable "scenario" { type = string }
variable "present" { type = bool }
variable "revision" { type = number }
resource "terraform_data" "flag" {
  input            = var.present
  triggers_replace = var.revision
}
resource "rediscloud_agent_memory" "example" {
  name                       = "unknown-block-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300

  dynamic "llm" {
    for_each = var.scenario == "models" ? (terraform_data.flag.output ? [true] : []) : (contains(["llm-credentials", "embedding-credentials"], var.scenario) ? [true] : [])
    content {
      provider = "openai"
      model    = "test-model"
      dynamic "credentials" {
        for_each = var.scenario == "llm-credentials" ? (terraform_data.flag.output ? [true] : []) : [true]
        content {
          type            = "api_key"
          api_key         = "FAKE-MODEL-SECRET"
          api_key_version = 1
        }
      }
    }
  }
  dynamic "embedding" {
    for_each = var.scenario == "models" ? (terraform_data.flag.output ? [true] : []) : (contains(["llm-credentials", "embedding-credentials"], var.scenario) ? [true] : [])
    content {
      provider = "openai"
      model    = "test-model"
      dynamic "credentials" {
        for_each = var.scenario == "embedding-credentials" ? (terraform_data.flag.output ? [true] : []) : [true]
        content {
          type            = "api_key"
          api_key         = "FAKE-MODEL-SECRET"
          api_key_version = 1
        }
      }
    }
  }
  dynamic "summarization" {
    for_each = var.scenario == "summarization" ? (terraform_data.flag.output ? [true] : []) : (var.scenario == "event-count" ? [true] : [])
    content {
      enabled = true
      dynamic "event_count" {
        for_each = var.scenario == "event-count" ? (terraform_data.flag.output ? [true] : []) : []
        content {
          threshold    = 20
          retain_count = 2
        }
      }
    }
  }
  dynamic "long_term_memory_exclusions" {
    for_each = var.scenario == "exclusions" ? (terraform_data.flag.output ? [true] : []) : (contains(["semantic", "built-in", "custom", "matcher", "regex"], var.scenario) ? [true] : [])
    content {
      enabled = true
      dynamic "semantic" {
        for_each = var.scenario == "semantic" ? (terraform_data.flag.output ? [true] : []) : []
        content {
          enabled = true
          prompt  = "Exclude secrets"
        }
      }
      dynamic "built_in_detectors" {
        for_each = var.scenario == "built-in" ? (terraform_data.flag.output ? [true] : []) : []
        content { enabled = true }
      }
      dynamic "custom_detectors" {
        for_each = var.scenario == "custom" ? (terraform_data.flag.output ? [true] : []) : (contains(["matcher", "regex"], var.scenario) ? [true] : [])
        content {
          enabled = true
          dynamic "detectors" {
            for_each = contains(["matcher", "regex"], var.scenario) ? [true] : []
            content {
              name    = "ticket"
              enabled = true
              action  = "redact"
              dynamic "matcher" {
                for_each = var.scenario == "matcher" ? (terraform_data.flag.output ? [true] : []) : [true]
                content {
                  kind = "regex"
                  dynamic "regex" {
                    for_each = var.scenario == "regex" ? (terraform_data.flag.output ? [true] : []) : [true]
                    content { pattern = "ticket-[0-9]+" }
                  }
                }
              }
            }
          }
        }
      }
    }
  }
  dynamic "custom_memory_types" {
    for_each = var.scenario == "extraction" ? [true] : []
    content {
      name        = "preference"
      description = "Durable preferences"
      fields {
        name        = "topic"
        description = "Preference topic"
        type        = "str"
      }
      dynamic "extraction_strategy" {
        for_each = terraform_data.flag.output ? [true] : []
        content {
          enabled = true
          prompt  = "Extract preferences"
        }
      }
    }
  }
  dynamic "timeouts" {
    for_each = var.scenario == "timeouts" ? (terraform_data.flag.output ? [true] : []) : []
    content { read = "7m" }
  }
}

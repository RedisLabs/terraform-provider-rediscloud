variable "custom_memory_types_description" {
  type    = string
  default = "User preferences"
}

variable "custom_memory_types_extraction_strategy_enabled" {
  type    = bool
  default = true
}

variable "custom_memory_types_extraction_strategy_prompt" {
  type    = string
  default = "Extract preferences"
}

variable "custom_memory_types_fields_0_description" {
  type    = string
  default = "Topic"
}

variable "custom_memory_types_fields_0_name" {
  type    = string
  default = "topic"
}

variable "custom_memory_types_fields_0_type" {
  type    = string
  default = "str"
}

variable "custom_memory_types_fields_1_description" {
  type    = string
  default = "Confidence"
}

variable "custom_memory_types_fields_1_name" {
  type    = string
  default = "confidence"
}

variable "custom_memory_types_fields_1_type" {
  type    = string
  default = "float"
}

variable "custom_memory_types_name" {
  type    = string
  default = "preference"
}

resource "rediscloud_agent_memory" "example" {
  name                       = "custom-type-store"
  database_id                = 123
  short_term_ttl_seconds     = 86400
  long_term_ttl_seconds      = 31536000
  extraction_cadence_seconds = 300
  custom_memory_types {
    name        = var.custom_memory_types_name
    description = var.custom_memory_types_description
    fields {
      name        = var.custom_memory_types_fields_0_name
      type        = var.custom_memory_types_fields_0_type
      description = var.custom_memory_types_fields_0_description
    }
    fields {
      name        = var.custom_memory_types_fields_1_name
      type        = var.custom_memory_types_fields_1_type
      description = var.custom_memory_types_fields_1_description
    }
    extraction_strategy {
      prompt  = var.custom_memory_types_extraction_strategy_prompt
      enabled = var.custom_memory_types_extraction_strategy_enabled
    }
  }


}

# variables.tf for inputs

variable "db_name" {
  type    = string
  default = "linknest"
}

variable "db_user" {
  type    = string
  default = "linknest"
}

variable "db_password" {
  type      = string
  sensitive = true
  # the sensitive flag hides value in plain output
}

variable "db_port" {
  type    = number
  default = 5434
}
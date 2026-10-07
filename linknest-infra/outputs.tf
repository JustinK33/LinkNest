# outputs.tf for outputs

output "postgres_host_port" {
  value = "localhost:${var.db_port}"
}

output "network_name" {
  value = docker_network.linknest.name
}

output "redis_host_port" {
  value = "localhost:${var.redis_port}"
}
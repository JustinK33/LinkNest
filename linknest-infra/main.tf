# main.tf for resources

# terraform fmt: rewrites the .tf files to follow standard spacing and alignment
# run it before every commit

# terraform validate: spell check for config. basically reads the code and checks if it make sense on its own: syntax, real arg names, refs that point at things that exists.
# needs init first bc it needs to know what arguments the provider allows

# terraform plan: the preview
# terraform does three things here
# 1. reads code (what you want)
# 2. reads the state file (what it remembers building)
# 3. asks the provider what exists right now
# then compares and prints what it would do

# '+' = create, '-' = destory, '~' = update in place

# terraform apply: does it for real. it runs the plan again, shows the same preview, then stops and asks: Do you want to perform these actions? Enter a value:
# type yes and it builds everything, in dependance order: network, volume, and image first, then the container
# when it finishes, it writes what it built into terraform.tfstate, then prints your outputs.

# good tip
# after running apply run plan again without changing anything and it should say "No changes. Your infrastructure matches the configuration." That message is Terraform confirming that code, state, and reality all agree.

resource "docker_network" "linknest" {
  name = "linknest_net"
}

resource "docker_volume" "pgdata" {
  name = "linknest-pgdata"
}

resource "docker_image" "postgres" {
  name = "postgres:16"
}

resource "docker_container" "postgres" {
  name  = "linknest_postgres"
  image = docker_image.postgres.image_id

  env = [
    "POSTGRES_DB=${var.db_name}",
    "POSTGRES_USER=${var.db_user}",
    "POSTGRES_PASSWORD=${var.db_password}",
  ]

  ports {
    internal = 5432
    external = var.db_port
  }

  volumes {
    volume_name    = docker_volume.pgdata.name
    container_path = "/var/lib/postgresql/data"
  }

  networks_advanced {
    name = docker_network.linknest.name
  }
}

resource "docker_image" "redis" {
  name = "redis:7"
}

resource "docker_volume" "redis_data" {
  name = "linknest-redis_data"
}

resource "docker_container" "redis" {
  name  = "linknest_redis"
  image = docker_image.redis.image_id

  ports {
    internal = 6379
    external = var.redis_port
  }

  volumes {
    volume_name    = docker_volume.redis_data.name
    container_path = "/data" # redis stores its data in /data
  }

  networks_advanced {
    name = docker_network.linknest.name
  }
}
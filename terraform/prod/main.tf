terraform {
  backend "s3" {
    bucket         = "terraform-state-storage-586877430255"
    dynamodb_table = "terraform-state-lock-586877430255"
    region         = "us-west-2"

    // THIS MUST BE UNIQUE
    key = "event-forwarder.tfstate"
  }
}

provider "aws" {
  region = "us-west-2"
}

data "aws_ssm_parameter" "eks_cluster_endpoint" {
  name = "/eks/av-cluster-endpoint"
}

provider "kubernetes" {
  host = data.aws_ssm_parameter.eks_cluster_endpoint.value
}

data "aws_ssm_parameter" "prd_db_addr" {
  name = "/env/couch-new-address"
}

data "aws_ssm_parameter" "prd_db_username" {
  name = "/env/couch-username"
}

data "aws_ssm_parameter" "prd_db_password" {
  name = "/env/couch-password"
}

data "aws_ssm_parameter" "elk_direct_address" {
  name = "/env/event-forwarder/elk-direct-address"
}

data "aws_ssm_parameter" "elk_username" {
  name = "/env/event-forwarder/elk-username"
}

data "aws_ssm_parameter" "elk_password" {
  name = "/env/event-forwarder/elk-password"
}

data "aws_ssm_parameter" "aws_access_key" {
  name = "/env/event-forwarder/aws-access-key"
}

data "aws_ssm_parameter" "aws_secret_key" {
  name = "/env/event-forwarder/aws-secret-key"
}

data "aws_ssm_parameter" "humio_direct_address" {
  name = "/env/event-forwarder/humio-direct-address"
}

module "event_forwarder" {
  //source = "github.com/byuoitav/terraform//modules/kubernetes-deployment"
  source = "github.com/byuoitav/terraform-pod-deployment//modules/kubernetes-deployment"

  // required
  name           = "event-forwarder"
  image          = "ghcr.io/byuoitav/event-forwarding-microservice/event-forwarding-microservice-amd64"
  image_version  = "v1.1.1"
  container_port = 8333
  repo_url       = "https://github.com/byuoitav/event-forwarding-microservice"

  // optional
  container_env = {
    "DB_ADDRESS"         = "https://${data.aws_ssm_parameter.prd_db_addr.value}",
    "DB_USERNAME"        = data.aws_ssm_parameter.prd_db_username.value,
    "DB_PASSWORD"        = data.aws_ssm_parameter.prd_db_password.value,
    "HUB_ADDRESS"        = "ws://event-hub"
    "STOP_REPLICATION"   = "true"
    "ELK_DIRECT_ADDRESS" = data.aws_ssm_parameter.elk_direct_address.value,
    "ELK_SA_USERNAME"    = data.aws_ssm_parameter.elk_username.value,
    "ELK_SA_PASSWORD"    = data.aws_ssm_parameter.elk_password.value,
    "AWS_SECRET_KEY"       = data.aws_ssm_parameter.aws_secret_key.value,
    "AWS_ACCESS_KEY"       = data.aws_ssm_parameter.aws_access_key.value,
    "HUMIO_DIRECT_ADDRESS" = data.aws_ssm_parameter.humio_direct_address.value,
  }
  container_args = []
  health_check   = false
}

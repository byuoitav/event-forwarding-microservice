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

module "event_forwarder" {
  source = "github.com/byuoitav/terraform//modules/kubernetes-deployment"

  // required
  name           = "event-forwarder"
  image          = "byuoitav/event-forwarding-microservice"
  image_version  = "development"
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
  }
  container_args = []
  health_check   = false
}

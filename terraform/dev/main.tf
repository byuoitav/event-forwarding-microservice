terraform {
  backend "s3" {
    bucket         = "terraform-state-storage-887007127029"
    dynamodb_table = "terraform-state-lock-887007127029"
    region         = "us-west-2"

    // THIS MUST BE UNIQUE
    key = "event-forwarder-dev.tfstate"
  }
}

provider "aws" {
  region = "us-west-2"
}

data "aws_ssm_parameter" "eks_cluster_endpoint" {
  name = "/eks/av-dev-cluster-endpoint"
}

provider "kubernetes" {
  host = data.aws_ssm_parameter.eks_cluster_endpoint.value
  config_path = "~/.kube/config"
}
data "aws_ssm_parameter" "prd_db_addr" {
  name = "/env/couch-address"
}

data "aws_ssm_parameter" "prd_db_username" {
  name = "/env/couch-username"
}

data "aws_ssm_parameter" "prd_db_password" {
  name = "/env/couch-password"
}

data "aws_ssm_parameter" "elk_direct_address" {
  name = "/env/event-forwarder-dev/elk-direct-address"
}

data "aws_ssm_parameter" "elk_username" {
  name = "/env/event-forwarder-dev/elk-username"
}

data "aws_ssm_parameter" "elk_password" {
  name = "/env/event-forwarder-dev/elk-password"
}

data "aws_ssm_parameter" "aws_access_key" {
  name = "/env/event-forwarder-dev/aws-access-key"
}

data "aws_ssm_parameter" "aws_secret_key" {
  name = "/env/event-forwarder-dev/aws-secret-key"
}

data "aws_ssm_parameter" "humio_direct_address" {
  name = "/env/event-forwarder-dev/humio-direct-address"
}

module "event_forwarder" {
  //source = "github.com/byuoitav/terraform//modules/kubernetes-deployment"
  source = "github.com/byuoitav/terraform-pod-deployment//modules/kubernetes-deployment"

  // required
  name           = "event-forwarder-dev"
  image          = "ghcr.io/byuoitav/event-forwarding-microservice/event-forwarding-microservice-amd64-dev"
  image_version  = "6c47825"
  container_port = 8333
  repo_url       = "https://github.com/byuoitav/event-forwarding-microservice"
  cluster        = "av-dev"
  environment    = "dev"
  route53_domain = "avdev.byu.edu"

  // optional
  container_env = {
    "DB_ADDRESS"         = "https://${data.aws_ssm_parameter.prd_db_addr.value}",
    "DB_USERNAME"        = data.aws_ssm_parameter.prd_db_username.value,
    "DB_PASSWORD"        = data.aws_ssm_parameter.prd_db_password.value,
    "HUB_ADDRESS"        = "ws://event-hub-dev"
    "STOP_REPLICATION"   = "true"
    "ELK_DIRECT_ADDRESS" = data.aws_ssm_parameter.elk_direct_address.value,
    "ELK_SA_USERNAME"    = data.aws_ssm_parameter.elk_username.value,
    "ELK_SA_PASSWORD"    = data.aws_ssm_parameter.elk_password.value,
    "AWS_SECRET_KEY"     = data.aws_ssm_parameter.aws_secret_key,
    "AWS_ACCESS_KEY"     = data.aws_ssm_parameter.aws_access_key,
    "HUMIO_DIRECT_ADDRESS" = data.aws_ssm_parameter.humio_direct_address,
  }
  container_args = []
  health_check   = false
}

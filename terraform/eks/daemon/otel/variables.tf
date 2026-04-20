// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

variable "region" {
  type    = string
  default = "us-west-2"
}

variable "test_dir" {
  type    = string
  default = "./test/otel/standard"
}

variable "cwagent_image_repo" {
  type    = string
  default = "public.ecr.aws/cloudwatch-agent/cloudwatch-agent"
}

variable "cwagent_image_tag" {
  type    = string
  default = "latest"
}

variable "helm_chart_branch" {
  type    = string
  default = "release-6.1.0"
}

variable "addon_version" {
  type    = string
  default = "v6.0.1-eksbuild.1"
}

variable "k8s_version" {
  type    = string
  default = "1.31"
}

variable "ami_type" {
  type    = string
  default = "AL2_x86_64"
}

variable "instance_type" {
  type    = string
  default = "t3.medium"
}

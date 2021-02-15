package main

import (
	_ "github.com/shalb/cluster.dev/pkg/backend/do"
	_ "github.com/shalb/cluster.dev/pkg/backend/s3"
	_ "github.com/shalb/cluster.dev/pkg/logging"
	_ "github.com/shalb/cluster.dev/pkg/modules/terraform/helm"
	_ "github.com/shalb/cluster.dev/pkg/modules/terraform/kubernetes"
	_ "github.com/shalb/cluster.dev/pkg/modules/terraform/tf_module"
	_ "github.com/shalb/cluster.dev/pkg/project"
	_ "github.com/shalb/cluster.dev/pkg/secrets/sops"
	_ "github.com/shalb/cluster.dev/pkg/secrets/ssm"
)
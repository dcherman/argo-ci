package main

import (
	"github.com/dcherman/argo-ci/cmd/github-token-sidecar/commands"
	"github.com/sirupsen/logrus"
)

func main() {
	cmd := commands.NewRootCommand()

	if err := cmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}

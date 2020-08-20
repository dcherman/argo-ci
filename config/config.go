package config

import (
	"time"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type Config struct {
	Builds []BuildConfig `yaml:"builds"`
}

type PullRequestConfig struct {
	AllowForks bool `yaml:"allowForks"`
}

type WorkflowConfig struct {
	File   *string        `yaml:"file,omitempty"`
	Source *wfv1.Workflow `yaml:"source,omitempty"`
}

type BuildConfig struct {
	Branches            []string          `yaml:"branches"`
	PullRequests        PullRequestConfig `yaml:"pullRequests"`
	Workflow            WorkflowConfig    `yaml:"workflow"`
	CommitStatusContext *string           `yaml:"commitStatusContext,omitempty"`
	TTL                 *time.Duration    `yaml:"ttl,omitempty"`
	Timeout             *time.Duration    `yaml:"timeout,omitempty"`

	// TODO: Secrets
}

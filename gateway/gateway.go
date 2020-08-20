package gateway

import (
	"fmt"
	"github.com/dcherman/argo-ci/config"
	"github.com/ghodss/yaml"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"regexp"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	argoclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
)
import "context"
import "github.com/google/go-github/v31/github"
import "encoding/json"

type Gateway struct {
 githubapp.ClientCreator

 argoclient argoclientset.Interface
 kubeclientset kubernetes.Interface
}

const (
	argoCiConfigPath = ".github/argo-ci.yaml"

	WorkflowLabelRepository = "argo-ci/repository"
	WorkflowLabelBuildRef = "argo-ci/git-ref"

	ArgoCiWorkflowTemplateName = "argo-ci-workflow"
)

var (
	headRegex = regexp.MustCompile(`refs/heads/(.+)`)
)

func (g *Gateway) Handles() []string {
	// Tag?
	return []string{"push", "pull_request"}
}

func (g *Gateway) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	switch eventType {
	case "push":
		var pushEvent github.PushEvent

		if err := json.Unmarshal(payload, &pushEvent); err != nil {
			return err
		}

		return g.HandlePushEvent(ctx, &pushEvent)
	}

	return fmt.Errorf("unknown event type: %s", eventType)
}

func (g *Gateway) GetConfig(ctx context.Context, client *github.Client, owner, repo, ref string) (*config.Config, error) {
	file, _, response, err := client.Repositories.GetContents(ctx, owner, repo, argoCiConfigPath, &github.RepositoryContentGetOptions{
		Ref: ref,
	})

	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected status code 200, got %d", response.StatusCode)
	}

	content, err := file.GetContent()

	if err != nil {
		return nil, err
	}

	var cfg config.Config

	if err := json.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func  (g *Gateway) HandlePushEvent(ctx context.Context, event *github.PushEvent) error {
	client, err := g.NewInstallationClient(*event.GetInstallation().ID)

	if err != nil {
		return err
	}

	owner := event.Repo.Owner.GetName()
	repo := event.Repo.GetName()
	ref := event.HeadCommit.GetID()

	config, err := g.GetConfig(ctx, client, owner, repo, ref)

	if err != nil {
		return err
	}

	for _, build := range config.Builds {
		if g.FilterPushEvent(ctx, event, &build) {
			if err := g.Dispatch(&build, event.Repo.GetURL(), ref); err != nil {
				logrus.Errorf("%v", err)
			}
		}
	}
}

func (g* Gateway) FilterPushEvent(ctx context.Context, event *github.PushEvent, buildConfig *config.BuildConfig) bool {
	ref := event.GetRef()
	match := headRegex.FindStringSubmatch(ref)

	if len(match) == 0 {
		return false
	}

	head := match[1]

	for _, branch := range buildConfig.Branches {
		if branch == head {
			return true
		}
	}

	return false
}

func (g *Gateway) CreateWorkflowNamespace(ctx context.Context, name string) error {
	return nil
}

// Dispatch
func (g *Gateway) Dispatch(config *config.BuildConfig, repoUrl, ref string) error {
	workflowBytes, err := yaml.Marshal(config.Workflow.Source)

	if err != nil {
		return err
	}

	workflowYaml := string(workflowBytes)

	workflow := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "TODO",

			Labels: map[string]string{
				WorkflowLabelRepository: repoUrl,
				WorkflowLabelBuildRef: ref,
			},
		},

		Spec: wfv1.WorkflowSpec{
			WorkflowTemplateRef:  &wfv1.WorkflowTemplateRef{
				Name: ArgoCiWorkflowTemplateName,
			},

			Arguments: wfv1.Arguments{
				Parameters: []wfv1.Parameter{
					{
						Name: "git-ref",
						Value: &ref,
					},
					{
						Name: "git-repo",
						Value: &repoUrl,
					},
					{
						Name: "workflow",
						Value: &workflowYaml,
					},
				},
			},
		},
	}

	created, err := g.argoclient.ArgoprojV1alpha1().Workflows("").Create(workflow)

	if err != nil {
		return err
	}

	logrus.Infof("created workflow %s/%s", created.Namespace, created.Name)
}


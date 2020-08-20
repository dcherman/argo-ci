package ci

import (
	"context"
	"fmt"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	WorkspaceClaimNameArgument = "workspaceClaimName"
	GitRefArgument = "git-ref"
	GitRepoArgument = "git-repo"

	WorkspaceVolumeName           = "workspace"
	DefaultWorkspaceMountLocation = "/home/argo-ci/workspace"
)

type WorkflowGenerator struct {
	kubeclientset  kubernetes.Interface
	sourceWorkflow *wfv1.Workflow
	ctx            context.Context

	gitRepoUrl string
	gitRef string

	ownerRef metav1.OwnerReference
}

func NewWorkflowGenerator() {

}

func SetWorkflowArguments(arguments *wfv1.Arguments, values map[string]string) {
	for k, v := range values {
		SetWorkflowArgument(arguments, CreateWorkflowArgument(k, v))
	}
}

func SetWorkflowArgument(arguments *wfv1.Arguments, parameter wfv1.Parameter) {
	for i := range arguments.Parameters {
		p := arguments.Parameters[i]

		if p.Name == parameter.Name {
			p.Value = parameter.Value
			p.ValueFrom = parameter.ValueFrom

			return
		}
	}

	arguments.Parameters = append(arguments.Parameters, parameter)
}

func CreateWorkflowArgument(name, value string) wfv1.Parameter {
	return wfv1.Parameter{
		Name:  name,
		Value: &value,
	}
}

func (wg *WorkflowGenerator) Render() (string, error) {
	// Add the owner reference to the input workflow so that when the current workflow is garbage collected,
	// our child workflow is as well
	wg.sourceWorkflow.OwnerReferences = append(wg.sourceWorkflow.OwnerReferences, wg.ownerRef)

	workspaceVolume, err := wg.GetSourcePVC("", "")

	if err != nil {
		return "", err
	}

	arguments := map[string]string{
		WorkspaceClaimNameArgument: workspaceVolume.PersistentVolumeClaim.ClaimName,
		GitRefArgument: wg.gitRef,
		GitRepoArgument: wg.gitRepoUrl,
	}

	SetWorkflowArguments(&wg.sourceWorkflow.Spec.Arguments, arguments)
}

func (wg *WorkflowGenerator) GetSourcePVC(thisNamespace, thisPodName string) (*corev1.Volume, error) {
	pod, err := wg.kubeclientset.CoreV1().Pods(thisNamespace).Get(wg.ctx, thisPodName, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	for _, volume := range pod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			return &volume, nil
		}
	}

	return nil, fmt.Errorf("no volume found")
}

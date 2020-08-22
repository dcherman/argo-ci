package tokenrefresh

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"
	"net/http"
	"sigs.k8s.io/yaml"
	"time"
)

type TokenRefresher struct {
	DryRun bool

	Output string

	RefreshInterval time.Duration

	GithubAppId    int64
	PrivateKeyPath string

	InstallationId int64

	GhUrl string

	PodName string
	PodUID  string

	SecretName      string
	SecretNamespace string
	SecretKey       string

	KubeClient kubernetes.Interface
}

var getSecretBackoff = wait.Backoff{
	Steps:    3,
	Duration: 500 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

func (r *TokenRefresher) NextToken(ctx context.Context) (string, error) {
	itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, r.GithubAppId, r.InstallationId, r.PrivateKeyPath)

	if err != nil {
		return "", err
	}

	return itr.Token(ctx)
}

func (r *TokenRefresher) Refresh(ctx context.Context) error {
	nextToken, err := r.NextToken(ctx)

	if err != nil {
		return err
	}

	secretsClient := r.KubeClient.CoreV1().Secrets(r.SecretNamespace)

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.SecretNamespace,
			Name:      r.SecretName,

			// Garbage collect this secret when this pod is deleted, meaning that this secret will be
			// indirectly deleted when the parent Workflow that resulted in its creation is deleted
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "apps/v1",
					Kind:               "Pod",
					BlockOwnerDeletion: pointer.BoolPtr(true),
					Name:               r.PodName,
					UID:                types.UID(r.PodUID),
				},
			},
		},
		Data: map[string][]byte{
			r.SecretKey: []byte(nextToken),
		},
	}

	switch r.Output {
	case "yaml":
		output, err := yaml.Marshal(secret)

		if err != nil {
			logrus.Errorf("failed to marshal secret to yaml: %v", err)
		}

		logrus.Infof("create or updating secret %s/%s: %s", r.SecretNamespace, r.SecretName, string(output))
	case "json":
		output, err := json.Marshal(secret)

		if err != nil {
			logrus.Errorf("failed to marshal secret to json: %v", err)
		}

		logrus.Infof("create or updating secret %s/%s: %s", r.SecretNamespace, r.SecretName, string(output))
	}

	var dryRunOpts []string

	if r.DryRun {
		dryRunOpts = append(dryRunOpts, metav1.DryRunAll)
	}

	_, err = secretsClient.Create(ctx, secret, metav1.CreateOptions{DryRun: dryRunOpts})

	if err != nil {
		if kubeerrors.IsAlreadyExists(err) {
			_, err := secretsClient.Update(ctx, secret, metav1.UpdateOptions{DryRun: dryRunOpts})

			return err
		}

		return err
	}

	return nil
}

func (r *TokenRefresher) Run(ctx context.Context) {
	for {
		var err error

		_ = wait.ExponentialBackoff(getSecretBackoff, func() (bool, error) {
			err = r.Refresh(ctx)

			if err != nil {
				if errors.Is(err, context.Canceled) {
					return true, nil
				}

				logrus.Errorf("failed to create or update secret %s/%s: %v", r.SecretNamespace, r.SecretName, err)
				return false, nil
			}

			return true, nil
		})

		nextInterval := r.RefreshInterval

		if err != nil {
			if !errors.Is(err, context.Canceled) {
				logrus.Errorf("failed to create/update our github token secret: %v.  trying again in 5s", err)
				nextInterval = time.Second * 5
			}
		} else {
			logrus.Infof("secret %s/%s successfully updated.  checking again in %s", r.SecretNamespace, r.SecretName, nextInterval)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(nextInterval):
			break
		}
	}
}

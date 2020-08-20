package tokenrefresh

import (
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"time"
)

type TokenRefresher struct {
	githubAppId string
	privateKey string

	secretName string
	secretNamespace string

	clientset kubernetes.Clientset
}

var getSecretBackoff = wait.Backoff{
	Steps:    3,
	Duration: 500 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

func (r *TokenRefresher) Run(doneCh chan <- struct{}) {
	for {
		var secret *corev1.Secret
		var err error

		_ = wait.ExponentialBackoff(getSecretBackoff, func() (bool, error) {
			secret, err = r.clientset.CoreV1().Secrets(r.secretNamespace).Get(r.secretName, metav1.GetOptions{})

			if err != nil {
				if errors.IsNotFound(err) {
					return true, err
				}

				logrus.Errorf("failed to get secret %s/%s: %v", r.secretNamespace ,r.secretName, err)

				return false, nil
			}

			return true, nil
		})

		if err

		// No need to sleep in the event that we encountered
		if err != nil {
			continue
		}
	}
}


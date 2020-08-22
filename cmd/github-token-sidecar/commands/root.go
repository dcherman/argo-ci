package commands

import (
	"context"
	"github.com/dcherman/argo-ci/tokenrefresh"
	"github.com/dcherman/argo-ci/util/kube"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func NewRootCommand() *cobra.Command {
	var (
		dryRun          bool
		secretName      string
		secretNamespace string
		secretKey       string
		podUID          string
		podName         string
		ghURL           string
		appId           int64
		installationId  int64
		privateKeyPath  string
		output          string

		refreshInterval time.Duration
	)

	rootCmd := &cobra.Command{
		Use:   "github-token-sidecar",
		Short: "Maintain a usable github token for a workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeClient, err := kube.NewKubeClientset()

			if err != nil {
				return err
			}

			rootContext, rootCanceler := context.WithCancel(context.Background())

			stopCh := make(chan os.Signal, 2)
			signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				signal := <-stopCh
				logrus.Infof("%s received.  exiting.", signal)
				rootCanceler()
			}()

			tokenRefresher := tokenrefresh.TokenRefresher{
				DryRun:          dryRun,
				SecretNamespace: secretNamespace,
				SecretName:      secretName,
				SecretKey:       secretKey,
				Output:          output,

				InstallationId: installationId,
				GithubAppId:    appId,

				KubeClient: kubeClient,

				PodUID:         podUID,
				PodName:        podName,
				PrivateKeyPath: privateKeyPath,

				GhUrl: ghURL,

				RefreshInterval: refreshInterval,
			}

			tokenRefresher.Run(rootContext)

			return nil
		},
	}

	flags := rootCmd.Flags()

	flags.StringVar(&secretName, "secret-name", "", "The name of the secret to create/update")
	flags.StringVar(&secretNamespace, "secret-namespace", "", "The namespace to create the secret in")
	flags.StringVar(&secretKey, "secret-key", "token", "The key in the secret to store the token")
	flags.StringVar(&podUID, "pod-uid", "", "The UID of the pod where this is running")
	flags.StringVar(&podName, "pod-name", "", "The name of the pod where this is running")
	flags.StringVar(&ghURL, "gh-url", "https://api.github.com", "The Github API URL to use for our requests")
	flags.StringVar(&privateKeyPath, "private-key-path", "", "The path to the private key for your github app")
	flags.DurationVar(&refreshInterval, "refresh-interval", time.Minute*45, "The interval at which to refresh the token")
	flags.Int64Var(&appId, "app-id", 0, "The ID of the Github Application")
	flags.Int64Var(&installationId, "installation-id", 0, "The ID of the Installation")
	flags.BoolVar(&dryRun, "dry-run", false, "Whether or not to actually create the secret")
	flags.StringVarP(&output, "output", "o", "", "If provided, output the secret to stdout in the given format")

	must(rootCmd.MarkFlagRequired("secret-name"))
	must(rootCmd.MarkFlagRequired("secret-namespace"))
	must(rootCmd.MarkFlagRequired("private-key-path"))
	must(rootCmd.MarkFlagRequired("pod-uid"))
	must(rootCmd.MarkFlagRequired("pod-name"))
	must(rootCmd.MarkFlagRequired("app-id"))
	must(rootCmd.MarkFlagRequired("installation-id"))

	return rootCmd
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

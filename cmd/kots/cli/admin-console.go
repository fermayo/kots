package cli

import (
	"os"
	"os/signal"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AdminConsoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "admin-console",
		Short:         "Make the admin console accessible",
		Long:          "Establish port forwarding for localhost access to the kotsadm admin console.",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			log := logger.NewLogger()

			clientset, err := k8sutil.GetClientset(kubernetesConfigFlags)
			if err != nil {
				return errors.Wrap(err, "failed get to get clientset")
			}

			podName, err := k8sutil.WaitForKotsadm(clientset, v.GetString("namespace"), time.Second*5)
			if err != nil {
				return errors.Wrap(err, "failed to wait for web")
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			adminConsolePort, errChan, err := k8sutil.PortForward(kubernetesConfigFlags, 8800, 3000, v.GetString("namespace"), podName, true, stopCh, log)
			if err != nil {
				return errors.Wrap(err, "failed to port forward")
			}

			go func() {
				select {
				case err := <-errChan:
					if err != nil {
						log.Error(err)
						os.Exit(-1)
					}
				case <-stopCh:
				}
			}()

			if adminConsolePort != 8800 {
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner("Port 8800 is not available. The Admin Console is running on port %d", adminConsolePort)
				log.ActionWithoutSpinner("")
			}

			log.ActionWithoutSpinner("Press Ctrl+C to exit")
			log.ActionWithoutSpinner("Go to http://localhost:%d to access the Admin Console", adminConsolePort)

			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt)

			<-signalChan

			log.ActionWithoutSpinner("Cleaning up")

			return nil
		},
	}

	cmd.AddCommand(AdminConsoleUpgradeCmd())

	return cmd
}

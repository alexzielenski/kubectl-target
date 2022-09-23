package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/component-base/cli"
)

var rootCmd *cobra.Command = &cobra.Command{
	Use:   "kubectl-credentials-local",
	Short: "kubectl-credentials-local is a generic unverified kubeconfig for local unsecured clusters",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(`apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority: /var/run/kubernetes/server-ca.crt
    server: https://localhost:6443
  name: local
contexts:
- context:
    cluster: local
    user: myself
  name: local
current-context: local
preferences: {}
users:
- name: myself
  user:
    client-certificate: /var/run/kubernetes/client-admin.crt
    client-key: /var/run/kubernetes/client-admin.key
`)
	},
}

func main() {
	os.Exit(cli.Run(rootCmd))
}

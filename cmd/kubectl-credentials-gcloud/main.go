package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"k8s.io/component-base/cli"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

var Flags struct {
	Project string
	Region  string
	Cluster string
}

var rootCmd *cobra.Command = &cobra.Command{
	Use:   "kubectl-credentials-gke",
	Short: "kubectl-credentials-gke is a generic unverified kubeconfig for gke lusters",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(Flags.Project) == 0 || len(Flags.Cluster) == 0 || len(Flags.Region) == 0 {
			fmt.Printf("missing flags")
			return
		}

		func() {
			file, err := os.CreateTemp("", "config")
			cmdutil.CheckErr(err)

			path := file.Name()
			defer os.Remove(file.Name())
			defer file.Close()

			cmd := exec.Command("gcloud")
			cmd.Args = append(cmd.Args, "get-credentials", "--region", Flags.Region, "--project", Flags.Project, "--cluster", Flags.Cluster)
			cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", path))

			cmd.Stderr = os.Stderr
			cmd.Stdout = nil

			cmd.Start()
			err = cmd.Wait()
			cmdutil.CheckErr(err)

			fmt.Print(ioutil.ReadAll(file))
		}()
	},
}

func init() {
	rootCmd.Flags().StringVar(&Flags.Project, "project", Flags.Project, "Project ID")
	rootCmd.Flags().StringVar(&Flags.Region, "region", Flags.Region, "Region ID ID")
	rootCmd.Flags().StringVar(&Flags.Cluster, "cluster", Flags.Cluster, "Cluster ID")
}

func main() {
	os.Exit(cli.Run(rootCmd))
}

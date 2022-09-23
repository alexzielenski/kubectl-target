package main

// Usage:
//
//	kubectl target gcloud project_id zone cluster -- apply -f XXX
//		Sends the specific operation to the given cluster
//
//		Skips/fails on conflicting/missing annotation?
//
//	kubectl target -- apply -f XXX
//
//  Intercepts input to `apply` commands. Interprets annotations such as:
//
//		annotations:
//			kubectl-target/provider: gke
//			kubectl-target/gke/project: $project
//			kubectl-target/gke/zone: $zone
//			kubectl-target/gke/cluster: $cluster
//
//	Directs into specific buckets
//
import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/component-base/cli"
	"k8s.io/kubectl/pkg/cmd/apply"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/yaml"
	// builder "/Users/alex/.asdf/installs/golang/1.19.1/packages/pkg/mod/k8s.io/cli-runtime@v0.25.2/pkg/resource/"
)

const KUBECTL_TARGET_ANNOTATION_PREFIX = "kubectl-target/"
const KUBECTL_TARGET_ANNOTATION_PROVIDER = KUBECTL_TARGET_ANNOTATION_PREFIX + "provider"

var rootCmd *cobra.Command = &cobra.Command{
	Use:   "kubectl-target",
	Short: "kubectl-target is an extensible way to target resources to a designated cluster",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		// This is the generic kubectl-target invocation.
		// This implementation is responsible for directing the input files to
		//	wherever they need to go for supported commands.
		//
		dashIndex := cmd.ArgsLenAtDash()

		if dashIndex < 0 {
			fmt.Printf(`error: missing '--'.
You must supply arguments to be forwarded to kubectl using '--'. Example:
		cat my_yaml.yml | kubectl-target -- apply -f -
`)
			return
		}

		// unforwardedArgs := args[:dashIndex]
		forwardedArgs := args[dashIndex:]

		runWithArgs := func(stdin []byte, args []string, outBuf *bytes.Buffer) {
			// No support for interpreting invocations to this command. Just
			// forward it as normal to kubectl
			cmd := exec.Command("kubectl")
			cmd.Args = append(cmd.Args, args...)
			if len(stdin) > 0 {
				cmd.Stdin = bytes.NewReader(stdin)
			}
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout

			if outBuf != nil {
				cmd.Stdout = outBuf
			}

			cmd.Start()
			err := cmd.Wait()

			if err != nil {
				// fmt.Print()

				// Use same exit code, if we can. If it is not available,
				// then there was an error running the process.
				if exitError, ok := err.(*exec.ExitError); ok {
					os.Exit(exitError.ExitCode())
				} else {
					cmdutil.CheckErr(err)
				}
			}
		}

		runOriginalCommand := func() { runWithArgs(nil, forwardedArgs, nil) }

		fetchKubeConfig := func(provider string, argMap map[string]string) (string, error) {
			args := []string{}
			args = append(args, "credentials")
			args = append(args, provider)

			for k, v := range argMap {
				args = append(args, "--"+k)
				args = append(args, v)
			}

			var buf bytes.Buffer
			runWithArgs(nil, args, &buf)
			return buf.String(), nil
		}

		// Parse flags for kubectl
		//
		iostreams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
		// kubeConfigFlags := genericclioptions.NewConfigFlags(true)
		// factory := cmdutil.NewFactory(kubeConfigFlags)

		// Create a stripped kubectl argument parser with only support for
		// apply subcommand
		kubectlCommand := &cobra.Command{}
		applyCommand := &cobra.Command{
			Use: "apply",
		}
		applyFlags := apply.NewApplyFlags(nil, iostreams)
		applyFlags.AddFlags(applyCommand)
		kubectlCommand.AddCommand(applyCommand)

		matched, flags, err := kubectlCommand.Find(forwardedArgs)
		if err != nil {
			// No matching subcommand. THis means we do not support custom
			// wrapping functionality of the kubectl command. Defer to
			// kubectl
			runOriginalCommand()
			return
		}

		err = matched.ParseFlags(flags)
		if err != nil {
			// Error parsing flags. Just defer to orignal command to show
			// correct error.
			runOriginalCommand()
			return
		}

		// Intercept all input resources to the apply command, and
		// put them into buckets according to their targetProvider/identifier
		//

		// If --prune is present in flags, defer to original kubectl since we
		// do not support prune here.
		if applyFlags.Prune {
			runOriginalCommand()
			return
		}

		filenameOptions := applyFlags.DeleteFlags.FileNameFlags.ToOptions()
		builder := resource.NewLocalBuilder()
		r := builder.
			Unstructured().
			Local().
			// Schema(o.Validator).
			ContinueOnError().
			// NamespaceParam(o.Namespace).DefaultNamespace().
			FilenameParam(false, &filenameOptions).
			// LabelSelectorParam(o.Selector).
			Flatten().
			Do()

		objects, err := r.Infos()
		cmdutil.CheckErr(err)

		// For each resource, group by annotation arguments.
		//
		// Map from provider name to map from key of that provider to its index
		//	used to generate keys for providerGroups
		providerGroupNames := map[string]map[string]int{}
		providerGroups := map[string]map[string][]runtime.Object{}

		registerGroupKey := func(provider, key string) int {
			groups, ok := providerGroupNames[provider]
			if !ok {
				groups = map[string]int{}
				providerGroupNames[provider] = groups
			}

			idx, ok := groups[key]
			if !ok {
				idx = len(groups)
				groups[key] = idx
			}

			return idx
		}

		registerGroup := func(provider string, options map[string]string, object runtime.Object) {
			groups, ok := providerGroups[provider]
			if !ok {
				groups = map[string][]runtime.Object{}
				providerGroups[provider] = groups
			}

			// Generate key from options
			args := make([]string, len(options), len(options))
			for k, v := range options {
				idx := registerGroupKey(provider, k)
				args[idx] = v
			}

			joinedKey := strings.Join(args, "/")

			group, ok := groups[joinedKey]
			if !ok {
				group = []runtime.Object{}
			}
			groups[joinedKey] = append(group, object)
		}

		for _, info := range objects {
			obj := info.Object

			accessor, err := meta.Accessor(obj)
			cmdutil.CheckErr(err)

			annotations := accessor.GetAnnotations()
			provider, hasProvider := annotations[KUBECTL_TARGET_ANNOTATION_PROVIDER]
			if !hasProvider {
				cmdutil.CheckErr(fmt.Errorf("%s is missing annotation: %s", info.Name, KUBECTL_TARGET_ANNOTATION_PROVIDER))
			}

			// Now that we have identified the provider, find any options that need
			// to be passed to the provider supplied in the form of innotations
			//	with name:
			// 	$prefix/$provider/$key
			keysPrefix := KUBECTL_TARGET_ANNOTATION_PREFIX + "/" + provider + "/"
			providerOptions := map[string]string{}
			for k, v := range annotations {
				if !strings.HasPrefix(k, keysPrefix) {
					continue
				}

				key := k[len(keysPrefix):]
				providerOptions[key] = v
			}

			registerGroup(provider, providerOptions, obj)
		}

		for provider, argGroups := range providerGroups {
			argNames := providerGroupNames[provider]
			idxToArgName := make([]string, len(argNames))
			for k, v := range argNames {
				idxToArgName[v] = k
			}

			for args, resourcesToApply := range argGroups {
				// Ask provider to give kubeconfig for given args
				//
				argMap := map[string]string{}
				splitArgs := strings.Split(args, "/")
				if len(args) > 0 {
					fmt.Println(args, splitArgs, len(splitArgs))
					for i, v := range splitArgs {
						argMap[idxToArgName[i]] = v
					}
				}

				kubeConfig, err := fetchKubeConfig(provider, argMap)
				cmdutil.CheckErr(err)

				filteredArgs := []string{
					forwardedArgs[0],
				}
				for i := 1; i+1 < len(forwardedArgs); i += 2 {
					// Filter -f, -k arguments out
					if forwardedArgs[i] == "-f" || forwardedArgs[i] == "-k" || forwardedArgs[i] == "--filename" || forwardedArgs[i] == "--kustomize" {
						continue
					} else if forwardedArgs[i] == "--kubeconfig" {
						continue
					}

					filteredArgs = append(filteredArgs, forwardedArgs[i])
					filteredArgs = append(filteredArgs, forwardedArgs[i+1])
				}

				filteredArgs = append(filteredArgs, "-f")
				filteredArgs = append(filteredArgs, "-")
				// Execute
				//		KUBE_CONFIG_DATA="" kubectl apply -f - ...<rest of args>
				//

				// Serialize arguments as JSON Array
				func() {
					var serialized bytes.Buffer
					for _, rsrc := range resourcesToApply {
						yamlSerialized, err := yaml.Marshal(rsrc)
						cmdutil.CheckErr(err)
						serialized.WriteString("---\n")
						serialized.Write(yamlSerialized)
					}

					file, err := os.CreateTemp("", "config")

					cmdutil.CheckErr(err)
					path := file.Name()
					defer os.Remove(path)

					file.WriteString(kubeConfig)
					file.Close()

					argsCopy := make([]string, 0, len(filteredArgs))
					argsCopy = append(argsCopy, "--kubeconfig")
					argsCopy = append(argsCopy, path)

					for _, v := range filteredArgs {
						argsCopy = append(argsCopy, v)
					}
					runWithArgs([]byte(serialized.String()), argsCopy, nil)
				}()
			}
		}

		// There may be multiple resources in each file...
		// Must be split

		// Just pipe the contents of the files via stdin
		//
		// Use kubectl's ApplyOptions to get the same set of objects
		// used by apply
	},
}

func main() {
	os.Exit(cli.Run(rootCmd))
}

package target

// import (
// 	"github.com/spf13/cobra"
// 	cmdutil "k8s.io/kubectl/pkg/cmd/util"
// 	"k8s.io/kubectl/pkg/scheme"
// 	"k8s.io/kubectl/pkg/validation"

// 	"k8s.io/cli-runtime/pkg/genericclioptions"
// 	"k8s.io/cli-runtime/pkg/printers"
// 	"k8s.io/cli-runtime/pkg/resource"
// 	"k8s.io/kubectl/pkg/util/i18n"
// )

// type TargetOptions struct {
// 	PrintFlags *genericclioptions.PrintFlags
// 	Printer    printers.ResourcePrinter

// 	OutputVersion string
// 	Namespace     string

// 	builder   func() *resource.Builder
// 	local     bool
// 	validator func() (validation.Schema, error)

// 	resource.FilenameOptions
// 	genericclioptions.IOStreams
// }

// func NewTargetOptions(ioStreams genericclioptions.IOStreams) *TargetOptions {
// 	return &TargetOptions{
// 		PrintFlags: genericclioptions.NewPrintFlags("converted").WithTypeSetter(scheme.Scheme).WithDefaultOutput("yaml"),
// 		IOStreams:  ioStreams,
// 	}
// }

// // NewCmdConvert creates a command object for the generic "convert" action, which
// // translates the config file into a given version.
// func NewCmdTarget(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) *cobra.Command {
// 	o := NewTargetOptions(ioStreams)

// 	cmd := &cobra.Command{
// 		Use:                   "Target -f FILENAME",
// 		DisableFlagsInUseLine: true,
// 		Short:                 i18n.T("Target config files between different API versions"),
// 		// Long:                  TargetLong,
// 		// Example:               TargetExample,
// 		Run: func(cmd *cobra.Command, args []string) {
// 			cmdutil.CheckErr(o.Complete(f, cmd))
// 			cmdutil.CheckErr(o.RunTarget())
// 		},
// 	}

// 	cmd.Flags().BoolVar(&o.local, "local", o.local, "If true, convert will NOT try to contact api-server but run locally.")
// 	cmd.Flags().StringVar(&o.OutputVersion, "output-version", o.OutputVersion, i18n.T("Output the formatted object with the given group version (for ex: 'extensions/v1beta1')."))
// 	o.PrintFlags.AddFlags(cmd)

// 	cmdutil.AddValidateFlags(cmd)
// 	cmdutil.AddFilenameOptionFlags(cmd, &o.FilenameOptions, "to need to get converted.")
// 	return cmd
// }

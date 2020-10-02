package shell

import (
	"fmt"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x/jx-secret/pkg/cmd/vault/wait"
	"github.com/jenkins-x/jx-secret/pkg/rootcmd"
	"github.com/jenkins-x/jx-secret/pkg/vaults/vaultcli"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Runs a shell so you can access the vault in a kubernetes cluster
`)

	cmdExample = templates.Examples(`
		%s vault shell
	`)
)

// Options the options for the command
type Options struct {
	wait.Options
	Shell     string
	ShellArgs []string
	Env       map[string]string
	NoWait    bool
}

// NewCmdVaultShell creates a command object for the command
func NewCmdVaultShell() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "shell",
		Short:   "Runs a shell so you can access the vault in a kubernetes cluster",
		Aliases: []string{"sh"},
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Shell, "shell", "s", "bash", "the command line shell to execute")
	cmd.Flags().StringArrayVarP(&o.ShellArgs, "args", "", nil, "the arguments to pass to the shell command")

	o.Options.AddFlags(cmd)
	return cmd, o
}

// WaitForVault waits for vault to be available
func (o *Options) WaitForVault() error {
	if o.NoWait {
		return nil
	}
	err := o.Options.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to wait for vault")
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate settings")
	}

	err = o.WaitForVault()
	if err != nil {
		return errors.Wrapf(err, "failed to wait for vault")
	}

	vaultBin, err := vaultcli.VerifyVaultBinary(o.CommandRunner, o.Env)
	if err != nil {
		return errors.Wrapf(err, "failed to validate vault binary")
	}

	if o.Env == nil {
		o.Env = map[string]string{}
	}
	env, err := vaultcli.CreateVaultEnv(o.KubeClient)
	if err != nil {
		return errors.Wrapf(err, "failed to setup the vault environment")
	}

	for k, v := range env {
		o.Env[k] = v
	}

	// lets add the vault binary to the PATH...
	log.Logger().Infof("using vault binary %s", vaultBin)

	// lets verify we can list the secrets
	cmd := &cmdrunner.Command{
		Name: o.Shell,
		Args: o.ShellArgs,
		Env:  o.Env,
	}
	_, err = o.CommandRunner(cmd)
	if err != nil {
		return errors.Wrapf(err, "failed to access vault. are you sure you are running the 'jx-secret vault portforward' command? command failed: %s", cmdrunner.CLI(cmd))
	}
	return nil

}

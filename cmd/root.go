package cmd

import (
	"e2e-k8s/internal/e2e"
	"github.com/spf13/cobra"
	"io"
)

var (
	ingressURL   string
	registryURL  string
	e2eNamespace string
	e2eJobName   string
)

const globalUsage = `
Run e2e testing against kubernetes vanilla
`

func NewRootCmd(_ io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "e2e",
		Short: "e2e testing kubernetes vanilla",
		Long:  globalUsage,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runE2e()
			if err != nil {
				return err
			}
			return nil
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&ingressURL, "ingress-url", "u", "", "Specify Ingress URL")
	cobra.MarkFlagRequired(flags, "ingress-url")
	flags.StringVarP(&registryURL, "registry", "r", "", "Specify docker registry")
	cobra.MarkFlagRequired(flags, "registry")
	flags.StringVarP(&e2eJobName, "e2e-job-name", "j", "", "Specify e2e job name ")
	cobra.MarkFlagRequired(flags, "e2e-job-name")
	flags.StringVarP(&e2eNamespace, "e2e-namespace", "n", "", "Specify e2e namespace")
	cobra.MarkFlagRequired(flags, "e2e-namespace")

	return cmd
}

func runE2e() error {
	client, err := e2e.NewKubeClient()
	if err != nil {
		return err
	}

	flow := &e2e.FlowTest{
		Registry: registryURL,
		Host:     ingressURL,
	}

	err = flow.CreateResources(client)
	if err != nil {
		return err
	}
	err = flow.ValidateFlowE2e(client, flow.Host, e2eNamespace, e2eJobName)
	if err != nil {
		return err
	}

	err = e2e.Cleaning(client)
	if err != nil {
		return err
	}

	return nil
}

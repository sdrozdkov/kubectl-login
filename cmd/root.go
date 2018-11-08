package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/sdrozdkov/kubectl-login/helpers"
	"github.com/sdrozdkov/kubectl-login/kubeconfig"
	"github.com/sdrozdkov/kubectl-login/oauthclient"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kubectl login [username]",
	Short: "Simple OIDC login plugin for kubectl",
	Args:  cobra.MinimumNArgs(1),
	Long:  `Authenticate for kubectl with OIDC from CLI`,
	Run: func(cmd *cobra.Command, args []string) {
		p := strconv.Itoa(port)

		kubeConf := kubeconfig.OIDCKubeConfig{}
		kubeConf.ReadConfig(args[0])

		app := oauthclient.App{}
		app.Init(&kubeConf, p)

		helpers.Openbrowser(fmt.Sprintf("http://localhost:%s/auth", p))
		app.Run(p)
	},
}

var port int

func init() {
	rootCmd.Flags().IntVarP(&port, "port", "p", 33768, "Specify custom port")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

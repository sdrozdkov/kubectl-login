package kubeconfig

import (
	"fmt"
	"log"
	"os"

	"k8s.io/client-go/tools/clientcmd"
)

type OIDCKubeConfig struct {
	Issuer       string
	RefreshToken string
	IDToken      string
	ClientID     string
	ClientSecret string
	username     string
}

func (c *OIDCKubeConfig) ReadConfig(username string) {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	rawConfig, _ := config.RawConfig()
	users := rawConfig.AuthInfos

	_, ok := users[username]
	if ok != true {
		log.Fatalf("User with username %s not found in kube config", username)
	}

	if users[username].AuthProvider == nil {
		log.Fatalln("No auth-provider assigned to user in kube config, auth-provider must be specified")
	}

	if users[username].AuthProvider.Name != "oidc" {
		log.Fatalln("Wrong auth-provider, name for auth-provider must be oidc")
	}

	oidcConf := users[username].AuthProvider.Config

	c.Issuer = oidcConf["idp-issuer-url"]
	c.ClientID = oidcConf["client-id"]
	c.ClientSecret = oidcConf["client-secret"]
	c.RefreshToken = oidcConf["redresh-token"]
	c.IDToken = oidcConf["id-token"]
	c.username = username
}

func (c *OIDCKubeConfig) WriteNewTokens() error {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	rawConfig, err := config.RawConfig()
	if err != nil {
		log.Fatalf("Some error when reading kube config")
	}
	rawConfig.AuthInfos[c.username].AuthProvider.Config["id-token"] = c.IDToken
	rawConfig.AuthInfos[c.username].AuthProvider.Config["refresh-token"] = c.RefreshToken

	err = clientcmd.WriteToFile(rawConfig, fmt.Sprintf("%s/.kube/config", os.Getenv("HOME")))
	if err != nil {
		log.Fatalf("Tokens not writed to file %v", err)
	}

	return nil
}

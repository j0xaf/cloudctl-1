package api

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Contexts contains all configuration contexts of cloudctl
type Contexts struct {
	CurrentContext  string `yaml:"current"`
	PreviousContext string `yaml:"previous"`
	Contexts        map[string]Context
}

// Context configure cloudctl behaviour
type Context struct {
	ApiURL       string  `yaml:"url"`
	IssuerURL    string  `yaml:"issuer_url"`
	IssuerType   string  `yaml:"issuer_type"`
	CustomScopes string  `yaml:"custom_scopes"`
	ClientID     string  `yaml:"client_id"`
	ClientSecret string  `yaml:"client_secret"`
	HMAC         *string `yaml:"hmac"`
}

var defaultCtx = Context{
	ApiURL:    "http://localhost:8080/cloud",
	IssuerURL: "http://localhost:8080/",
}

func GetContexts() (*Contexts, error) {
	var ctxs Contexts
	cfgFile := viper.GetViper().ConfigFileUsed()
	c, err := os.ReadFile(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read config, please create a config.yaml in either: /etc/cloudctl/, $HOME/.cloudctl/ or in the current directory, see cloudctl ctx -h for examples")
	}
	err = yaml.Unmarshal(c, &ctxs)
	return &ctxs, err
}

func WriteContexts(ctxs *Contexts) error {
	c, err := yaml.Marshal(ctxs)
	if err != nil {
		return err
	}
	cfgFile := viper.GetViper().ConfigFileUsed()
	err = os.WriteFile(cfgFile, c, 0600)
	if err != nil {
		return err
	}
	fmt.Printf("%s switched context to \"%s\"\n", color.GreenString("✔"), color.GreenString(ctxs.CurrentContext))
	return nil
}

func MustDefaultContext() Context {
	ctxs, err := GetContexts()
	if err != nil {
		return defaultCtx
	}
	ctx, ok := ctxs.Contexts[ctxs.CurrentContext]
	if !ok {
		return defaultCtx
	}
	return ctx
}

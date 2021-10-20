package helper

import (
	"fmt"

	"github.com/metal-stack/metal-lib/auth"
	"gopkg.in/yaml.v3"
)

func EnrichKubeconfigTpl(tpl string, authContext *auth.AuthContext) ([]byte, error) {
	cfg := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(tpl), cfg)
	if err != nil {
		return nil, err
	}
	// identify clustername
	clusterNames, err := auth.GetClusterNames(cfg)
	if err != nil {
		return nil, err
	}
	if len(clusterNames) != 1 {
		return nil, fmt.Errorf("expected one cluster in config, got %d", len(clusterNames))
	}

	userName := authContext.User
	clusterName := clusterNames[0]
	contextName := fmt.Sprintf("%s@%s", userName, clusterName)

	// merge with current user credentials
	err = auth.AddUser(cfg, *authContext)
	if err != nil {
		return nil, err
	}
	err = auth.AddContext(cfg, contextName, clusterName, userName)
	if err != nil {
		return nil, err
	}
	auth.SetCurrentContext(cfg, contextName)

	mergedKubeconfig, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	return mergedKubeconfig, nil
}

func MergeKubeconfigTpl(currentCfg map[interface{}]interface{}, tpl, contextName, clusterName string, authContext *auth.AuthContext) ([]byte, error) {
	clusters := &struct {
		Clusters []struct {
			Cluster map[string]interface{} `yaml:"cluster"`
		} `yaml:"clusters"`
	}{}

	err := yaml.Unmarshal([]byte(tpl), clusters)
	if err != nil {
		return nil, err
	}
	if len(clusters.Clusters) == 0 {
		return nil, fmt.Errorf("kubeconfig template from cloud-api does not contain cluster entry")
	}

	err = auth.AddCluster(currentCfg, clusterName, clusters.Clusters[0].Cluster)
	if err != nil {
		return nil, err
	}
	err = auth.AddUser(currentCfg, *authContext)
	if err != nil {
		return nil, err
	}
	err = auth.AddContext(currentCfg, contextName, clusterName, authContext.User)
	if err != nil {
		return nil, err
	}

	mergedKubeconfig, err := yaml.Marshal(currentCfg)
	if err != nil {
		return nil, err
	}

	return mergedKubeconfig, nil
}

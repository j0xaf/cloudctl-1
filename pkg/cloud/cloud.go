package cloud

import (
	"fmt"
	"net/url"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/metal-pod/security"

	"git.f-i-ts.de/cloud-native/cloudctl/api/client"
	"git.f-i-ts.de/cloud-native/cloudctl/api/client/billing"
	"git.f-i-ts.de/cloud-native/cloudctl/api/client/cluster"
	"git.f-i-ts.de/cloud-native/cloudctl/api/client/project"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// Cloud provides cloud functions
type Cloud struct {
	Cluster *cluster.Client
	Project *project.Client
	Billing *billing.Client
	Auth    runtime.ClientAuthInfoWriter
}

// NewCloud create a new Cloud
func NewCloud(apiurl, apiToken string) (*Cloud, error) {

	parsedurl, err := url.Parse(apiurl)
	if err != nil {
		return nil, err
	}
	if parsedurl.Host == "" {
		return nil, fmt.Errorf("invalid url:%s, must be in the form scheme://host[:port]/basepath", apiurl)
	}

	auther := runtime.ClientAuthInfoWriterFunc(func(rq runtime.ClientRequest, rg strfmt.Registry) error {
		if apiToken != "" {
			security.AddUserTokenToClientRequest(rq, apiToken)
		}
		return nil
	})

	transport := httptransport.New(parsedurl.Host, parsedurl.Path, []string{parsedurl.Scheme})
	transport.DefaultAuthentication = auther

	cloud := client.New(transport, strfmt.Default)

	c := &Cloud{
		Auth:    auther,
		Cluster: cloud.Cluster,
		Project: cloud.Project,
		Billing: cloud.Billing,
	}
	return c, nil
}

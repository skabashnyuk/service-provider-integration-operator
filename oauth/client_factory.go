// Copyright (c) 2021 Red Hat, Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	cli "github.com/redhat-appstudio/service-provider-integration-operator/cmd/oauth/oauthcli"
	"github.com/redhat-appstudio/service-provider-integration-operator/pkg/spi-shared/httptransport"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/codeready-toolchain/api/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/redhat-appstudio/service-provider-integration-operator/api/v1beta1"
	authz "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	certutil "k8s.io/client-go/util/cert"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8sClientFactoryBuilder struct {
	Args cli.OAuthServiceCliArgs
}

func (r K8sClientFactoryBuilder) CreateInClusterClientFactory() (clientFactory K8sClientFactory, err error) {
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{})
	mapper.Add(corev1.SchemeGroupVersion.WithKind("Secret"), meta.RESTScopeNamespace)
	clientOptions, errClientOptions := clientOptions(mapper)
	if errClientOptions != nil {
		return nil, errClientOptions
	}
	if r.Args.KubeConfig != "" {
		restConfig, err := clientcmd.BuildConfigFromFlags("", r.Args.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create rest configuration: %w", err)
		}
		return &InClusterK8sClientFactory{ClientOptions: clientOptions, RestConfig: restConfig}, nil
	}
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize in-cluster config: %w", err)
	}
	return &InClusterK8sClientFactory{ClientOptions: clientOptions, RestConfig: restConfig}, nil

}
func (r K8sClientFactoryBuilder) CreateUserAuthClientFactory() (clientFactory K8sClientFactory, err error) {
	// we can't use the default dynamic rest mapper, because we don't have a token that would enable us to connect
	// to the cluster just yet. Therefore, we need to list all the resources that we are ever going to query using our
	// client here thus making the mapper not reach out to the target cluster at all.
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{})
	mapper.Add(authz.SchemeGroupVersion.WithKind("SelfSubjectAccessReview"), meta.RESTScopeRoot)
	mapper.Add(v1beta1.GroupVersion.WithKind("SPIAccessToken"), meta.RESTScopeNamespace)
	mapper.Add(v1beta1.GroupVersion.WithKind("SPIAccessTokenDataUpdate"), meta.RESTScopeNamespace)
	clientOptions, errClientOptions := clientOptions(mapper)
	if errClientOptions != nil {
		return nil, errClientOptions
	}

	// here we're essentially replicating what is done in rest.InClusterConfig() but we're using our own
	// configuration - this is to support going through an alternative API server to the one we're running with...
	// Note that we're NOT adding the Token or the TokenFile to the configuration here. This is supposed to be
	// handled on per-request basis...
	cfg := &rest.Config{}

	apiServerUrl, err := url.Parse(r.Args.ApiServer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the API server URL: %w", err)
	}

	cfg.Host = "https://" + apiServerUrl.Host

	tlsConfig := rest.TLSClientConfig{}

	if r.Args.ApiServerCAPath != "" {
		// rest.InClusterConfig is doing this most possibly only for early error handling so let's do the same
		if _, err := certutil.NewPool(r.Args.ApiServerCAPath); err != nil {
			return nil, fmt.Errorf("expected to load root CA config from %s, but got err: %w", r.Args.ApiServerCAPath, err)
		} else {
			tlsConfig.CAFile = r.Args.ApiServerCAPath
		}
	}

	cfg.TLSClientConfig = tlsConfig

	if r.Args.ApiServer != "" {
		return &WorkspaceAwareK8sClientFactory{
			ClientOptions: clientOptions,
			RestConfig:    cfg,
			ApiServer:     r.Args.ApiServer,
			HTTPClient: &http.Client{
				Transport: httptransport.HttpMetricCollectingRoundTripper{
					RoundTripper: http.DefaultTransport}}}, nil
	}
	return &UserAuthK8sClientFactory{ClientOptions: clientOptions, RestConfig: cfg}, nil
}

type K8sClientFactory interface {
	CreateClient(ctx context.Context) (client.Client, error)
}

type WorkspaceAwareK8sClientFactory struct {
	ClientOptions *client.Options
	RestConfig    *rest.Config
	ApiServer     string
	HTTPClient    rest.HTTPClient
}

func (w WorkspaceAwareK8sClientFactory) CreateClient(ctx context.Context) (client.Client, error) {
	namespace := ctx.Value("namespace")
	if namespace != "" {
		lg := log.FromContext(ctx)
		wsEndpoint := path.Join(w.ApiServer, "apis/toolchain.dev.openshift.com/v1alpha1/workspaces") //TODO: configurable path ?
		req, reqErr := http.NewRequestWithContext(ctx, "GET", wsEndpoint, nil)
		if reqErr != nil {
			lg.Error(reqErr, "failed to create request for the workspace API", "url", wsEndpoint)
			return nil, fmt.Errorf("error while constructing HTTP request for workspace context to %s: %w", wsEndpoint, reqErr)
		}
		resp, err := w.HTTPClient.Do(req)
		if err != nil {
			lg.Error(err, "failed to request the workspace API", "url", wsEndpoint)
			return nil, fmt.Errorf("error performing HTTP request for workspace context to %v: %w", wsEndpoint, err)
		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				lg.Error(err, "Failed to close response body doing workspace fetch")
			}
		}()

		if resp.StatusCode == http.StatusOK {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Print(err.Error())
			}
			wsList := &v1alpha1.WorkspaceList{}
			json.Unmarshal(bodyBytes, wsList)
			for _, ws := range wsList.Items {
				for _, ns := range ws.Status.Namespaces {
					if ns.Name == namespace {
						w.RestConfig.APIPath = path.Join("workspaces", ws.Name)
						cl, err := client.New(w.RestConfig, *w.ClientOptions)
						if err != nil {
							return nil, fmt.Errorf("failed to create a kubernetes client: %w", err)
						}
						return cl, nil
					}
				}
			}
			return nil, fmt.Errorf("target workspace not found for namespace %s", namespace)
		} else {
			lg.Info("unexpected return code for workspace api", "url", wsEndpoint, "code", resp.StatusCode)
			return nil, fmt.Errorf("bad status (%d) when performing HTTP request for workspace context to %v: %w", resp.StatusCode, wsEndpoint, err)
		}
	}
	cl, err := client.New(w.RestConfig, *w.ClientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create a kubernetes client: %w", err)
	}
	return cl, nil
}

type UserAuthK8sClientFactory struct {
	ClientOptions *client.Options
	RestConfig    *rest.Config
}

func (u UserAuthK8sClientFactory) CreateClient(ctx context.Context) (client.Client, error) {
	cl, err := client.New(u.RestConfig, *u.ClientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create a kubernetes client: %w", err)
	}
	return cl, nil
}

type InClusterK8sClientFactory struct {
	ClientOptions *client.Options
	RestConfig    *rest.Config
}

func (i InClusterK8sClientFactory) CreateClient(ctx context.Context) (client.Client, error) {

	cl, err := client.New(i.RestConfig, *i.ClientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create a kubernetes client: %w", err)
	}
	return cl, nil
}

func clientOptions(mapper meta.RESTMapper) (*client.Options, error) {
	options := &client.Options{
		Mapper: mapper,
		Scheme: runtime.NewScheme(),
	}

	if err := corev1.AddToScheme(options.Scheme); err != nil {
		return nil, fmt.Errorf("failed to add corev1 to scheme: %w", err)
	}

	if err := v1beta1.AddToScheme(options.Scheme); err != nil {
		return nil, fmt.Errorf("failed to add api to the scheme: %w", err)
	}

	if err := authz.AddToScheme(options.Scheme); err != nil {
		return nil, fmt.Errorf("failed to add authz to the scheme: %w", err)
	}

	return options, nil
}

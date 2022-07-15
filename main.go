/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"net/http"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/redhat-appstudio/service-provider-integration-operator/pkg/logs"
	"github.com/redhat-appstudio/service-provider-integration-operator/pkg/serviceprovider"
	"github.com/redhat-appstudio/service-provider-integration-operator/pkg/serviceproviders"
	"github.com/redhat-appstudio/service-provider-integration-operator/pkg/spi-shared/tokenstorage"
	corev1 "k8s.io/api/core/v1"

	"github.com/redhat-appstudio/service-provider-integration-operator/controllers"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	appstudiov1beta1 "github.com/redhat-appstudio/service-provider-integration-operator/api/v1beta1"
	"github.com/redhat-appstudio/service-provider-integration-operator/pkg/config"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	//+kubebuilder:scaffold:imports

	sharedConfig "github.com/redhat-appstudio/service-provider-integration-operator/pkg/spi-shared/config"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(appstudiov1beta1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {

	args := config.SpiOperatorConfiguration{}
	arg.MustParse(&args)
	logs.InitLoggers(args.ZapDevel, args.ZapEncoder, args.ZapLogLevel, args.ZapStackTraceLevel, args.ZapTimeEncoding)
	setupLog := ctrl.Log.WithName("setup")
	setupLog.Info("Starting SPI operator with environment", "env", os.Environ(), "configuration", &args)
	if err := config.ValidateEnv(); err != nil {
		setupLog.Error(err, "invalid configuration")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     args.MetricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: args.ProbeAddr,
		LeaderElection:         args.EnableLeaderElection,
		LeaderElectionID:       "f5c55e16.appstudio.redhat.org",
		Logger:                 ctrl.Log,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	cfg, err := sharedConfig.LoadFrom(args.ConfigFile)
	if err != nil {
		setupLog.Error(err, "Failed to load the configuration")
		os.Exit(1)
	}

	strg, err := tokenstorage.NewVaultStorage("spi-controller-manager", cfg.VaultHost, cfg.ServiceAccountTokenFilePath, args.VaultInsecureTLS)
	if err != nil {
		setupLog.Error(err, "failed to initialize the token storage")
		os.Exit(1)
	}

	if config.RunControllers() {
		if err = (&controllers.SPIAccessTokenReconciler{
			Client:       mgr.GetClient(),
			Scheme:       mgr.GetScheme(),
			TokenStorage: strg,
			ServiceProviderFactory: serviceprovider.Factory{
				Configuration:    cfg,
				KubernetesClient: mgr.GetClient(),
				HttpClient:       http.DefaultClient,
				Initializers:     serviceproviders.KnownInitializers(),
				TokenStorage:     strg,
			},
			Configuration: cfg,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "SPIAccessToken")
			os.Exit(1)
		}
		if err = (&controllers.SPIAccessTokenBindingReconciler{
			Client:       mgr.GetClient(),
			Scheme:       mgr.GetScheme(),
			TokenStorage: strg,
			ServiceProviderFactory: serviceprovider.Factory{
				Configuration:    cfg,
				KubernetesClient: mgr.GetClient(),
				HttpClient:       http.DefaultClient,
				Initializers:     serviceproviders.KnownInitializers(),
				TokenStorage:     strg,
			},
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "SPIAccessTokenBinding")
			os.Exit(1)
		}
	} else {
		setupLog.Info("CRD controllers inactive")
	}

	if err = (&controllers.SPIAccessCheckReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		ServiceProviderFactory: serviceprovider.Factory{
			Configuration:    cfg,
			KubernetesClient: mgr.GetClient(),
			HttpClient:       http.DefaultClient,
			Initializers:     serviceproviders.KnownInitializers(),
			TokenStorage:     strg,
		},
		Configuration: cfg,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SPIAccessCheck")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

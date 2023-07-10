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

package v1beta1

import (
	"fmt"

	rapi "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SPIAccessTokenUploadSpec defines the desired state of SPIAccessTokenUpload
type SPIAccessTokenUploadSpec struct {
	// RepoUrl is just the URL of the repository for which the access token is requested.
	RepoUrl string `json:"repoUrl"`
	// Secret is the specification of the secret that should contain the access token.
	// The secret will be created in the same namespace as this binding object.
	Secret rapi.LinkableSecretSpec `json:"secret"`
	// +optional
	OAuth OAuthSpec `json:"oauth"`
	// Lifetime specifies how long the binding and its associated data should live.
	// This is specified as time with a unit (30m, 2h). A special value of "-1" means
	// infinite lifetime.
	Lifetime string `json:"lifetime,omitempty"`
}

type OAuthSpec struct {
	// Scopes are the list of OAuth scopes that this token possesses
	// +optional
	Scopes []string `json:"scopes"`
}

// SPIAccessTokenUploadStatus defines the observed state of SPIAccessTokenUpload
type SPIAccessTokenUploadStatus struct {
	Phase                 SPIAccessTokenUploadPhase       `json:"phase"`
	ErrorReason           SPIAccessTokenUploadErrorReason `json:"errorReason,omitempty"`
	ErrorMessage          string                          `json:"errorMessage,omitempty"`
	LinkedAccessTokenName string                          `json:"linkedAccessTokenName"`
	OAuthUrl              string                          `json:"oAuthUrl,omitempty"`
	UploadUrl             string                          `json:"uploadUrl,omitempty"`
	SyncedObjectRef       TargetObjectRef                 `json:"syncedObjectRef"`
}

type SPIAccessTokenUploadPhase string

const (
	SPIAccessTokenUploadPhaseAwaitingTokenData SPIAccessTokenUploadPhase = "AwaitingTokenData"
	SPIAccessTokenUploadPhaseInjected          SPIAccessTokenUploadPhase = "Injected"
	SPIAccessTokenUploadPhaseError             SPIAccessTokenUploadPhase = "Error"
)

type SPIAccessTokenUploadErrorReason string

const (
	SPIAccessTokenUploadErrorReasonUnknownServiceProviderType        SPIAccessTokenUploadErrorReason = "UnknownServiceProviderType"
	SPIAccessTokenUploadErrorUnsupportedServiceProviderConfiguration SPIAccessTokenUploadErrorReason = "UnsupportedServiceProviderConfiguration"
	SPIAccessTokenUploadErrorReasonInvalidLifetime                   SPIAccessTokenUploadErrorReason = "InvalidLifetime"
	SPIAccessTokenUploadErrorReasonTokenLookup                       SPIAccessTokenUploadErrorReason = "TokenLookup"
	SPIAccessTokenUploadErrorReasonLinkedToken                       SPIAccessTokenUploadErrorReason = "LinkedToken"
	SPIAccessTokenUploadErrorReasonTokenRetrieval                    SPIAccessTokenUploadErrorReason = "TokenRetrieval"
	SPIAccessTokenUploadErrorReasonTokenSync                         SPIAccessTokenUploadErrorReason = "TokenSync"
	SPIAccessTokenUploadErrorReasonTokenAnalysis                     SPIAccessTokenUploadErrorReason = "TokenAnalysis"
	SPIAccessTokenUploadErrorReasonUnsupportedPermissions            SPIAccessTokenUploadErrorReason = "UnsupportedPermissions"
	SPIAccessTokenUploadErrorReasonInconsistentSpec                  SPIAccessTokenUploadErrorReason = "InconsistentSpec"
	SPIAccessTokenUploadErrorReasonServiceAccountUnavailable         SPIAccessTokenUploadErrorReason = "ServiceAccountUnavailable"
	SPIAccessTokenUploadErrorReasonServiceAccountUpdate              SPIAccessTokenUploadErrorReason = "ServiceAccountUpdate"
	SPIAccessTokenUploadErrorReasonNoError                           SPIAccessTokenUploadErrorReason = ""
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SPIAccessTokenUpload is the Schema for the SPIAccessTokenUploads API
type SPIAccessTokenUpload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SPIAccessTokenUploadSpec   `json:"spec,omitempty"`
	Status SPIAccessTokenUploadStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SPIAccessTokenUploadList contains a list of SPIAccessTokenUpload
type SPIAccessTokenUploadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SPIAccessTokenUpload `json:"items"`
}

type SPIAccessTokenUploadValidation struct {
	// Consistency is the list of consistency validation errors
	Consistency []string
}

func init() {
	SchemeBuilder.Register(&SPIAccessTokenUpload{}, &SPIAccessTokenUploadList{})
}

func (in *SPIAccessTokenUpload) RepoUrl() string {
	return in.Spec.RepoUrl
}

func (in *SPIAccessTokenUpload) ObjNamespace() string {
	return in.Namespace
}

func (in *SPIAccessTokenUpload) Validate() SPIAccessTokenUploadValidation {
	ret := SPIAccessTokenUploadValidation{}

	for i, link := range in.Spec.Secret.LinkedTo {
		if link.ServiceAccount.Reference.Name != "" && (link.ServiceAccount.Managed.Name != "" || link.ServiceAccount.Managed.GenerateName != "") {
			ret.Consistency = append(ret.Consistency, fmt.Sprintf("The %d-th service account spec defines both a service account reference and the managed service account. This is invalid", i+1))
		}
		if in.Spec.Secret.Type != corev1.SecretTypeDockerConfigJson && link.ServiceAccount.As == rapi.ServiceAccountLinkTypeImagePullSecret {
			ret.Consistency = append(ret.Consistency,
				fmt.Sprintf("the secret must have the %s type for it to be linkable to the %d-th service account spec as an image pull secret", corev1.SecretTypeDockerConfigJson, i+1))
		}
	}

	return ret
}

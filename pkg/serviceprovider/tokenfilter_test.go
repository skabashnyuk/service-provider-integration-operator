//
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

package serviceprovider

import (
	"context"
	"fmt"
	"testing"

	"github.com/redhat-appstudio/remote-secret/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/redhat-appstudio/service-provider-integration-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
)

var filterTrue = TokenFilterFunc(func(ctx context.Context, binding Matchable, token *api.SPIAccessToken) (bool, error) {
	return true, nil
})
var filterFalse = TokenFilterFunc(func(ctx context.Context, binding Matchable, token *api.SPIAccessToken) (bool, error) {
	return false, nil
})

var filterError = TokenFilterFunc(func(ctx context.Context, binding Matchable, token *api.SPIAccessToken) (bool, error) {
	return false, fmt.Errorf("some error")
})

var conditionTrue = func() bool {
	return true
}
var conditionFalse = func() bool {
	return false
}

func TestTokenFilterFunc_Matches(t *testing.T) {
	type args struct {
		ctx       context.Context
		matchable Matchable
		token     *api.SPIAccessToken
	}
	tests := []struct {
		name    string
		f       TokenFilterFunc
		args    args
		want    bool
		wantErr assert.ErrorAssertionFunc
	}{
		{"test true filter", filterTrue, args{context.TODO(), &api.SPIAccessTokenBinding{}, &api.SPIAccessToken{}}, true, assert.NoError},
		{"test false filter", filterFalse, args{context.TODO(), &api.SPIAccessTokenBinding{}, &api.SPIAccessToken{}}, false, assert.NoError},
		{"test error filter", filterError, args{context.TODO(), &api.SPIAccessTokenBinding{}, &api.SPIAccessToken{}}, false, assert.Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.f.Matches(tt.args.ctx, tt.args.matchable, tt.args.token)
			if !tt.wantErr(t, err, fmt.Sprintf("Matches(%v, %v, %v)", tt.args.ctx, tt.args.matchable, tt.args.token)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Matches(%v, %v, %v)", tt.args.ctx, tt.args.matchable, tt.args.token)
		})
	}
}

func TestDefaultRemoteSecretFilterFunc(t *testing.T) {
	remoteSecret := v1beta1.RemoteSecret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs",
			Namespace: "ns",
		},
		Spec: v1beta1.RemoteSecretSpec{
			Secret: v1beta1.LinkableSecretSpec{
				Name: "secret",
				Type: corev1.SecretTypeBasicAuth,
			},
			Targets: []v1beta1.RemoteSecretTarget{{
				Namespace: "ns",
			}},
		},
		Status: v1beta1.RemoteSecretStatus{
			Conditions: []metav1.Condition{{
				Type:   string(v1beta1.RemoteSecretConditionTypeDataObtained),
				Status: metav1.ConditionTrue,
			}},
			Targets: []v1beta1.TargetStatus{{
				Namespace:  "ns",
				SecretName: "secret",
			}},
		},
	}
	accessCheck := api.SPIAccessCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "accessCheck",
			Namespace: "ns",
		},
	}

	t.Run("all conditions satisfied", func(t *testing.T) {
		assert.True(t, DefaultRemoteSecretFilterFunc(context.TODO(), &accessCheck, &remoteSecret))
	})

	t.Run("data not obtained", func(t *testing.T) {
		rs := remoteSecret.DeepCopy()
		rs.Status.Conditions[0].Status = metav1.ConditionFalse
		assert.False(t, DefaultRemoteSecretFilterFunc(context.TODO(), &accessCheck, rs))
	})

	t.Run("wrong secret type", func(t *testing.T) {
		rs := remoteSecret.DeepCopy()
		rs.Spec.Secret.Type = corev1.SecretTypeOpaque
		assert.False(t, DefaultRemoteSecretFilterFunc(context.TODO(), &accessCheck, rs))
	})

	t.Run("target in different namespace", func(t *testing.T) {
		rs := remoteSecret.DeepCopy()
		rs.Status.Targets[0].Namespace = "diff-ns"
		assert.False(t, DefaultRemoteSecretFilterFunc(context.TODO(), &accessCheck, rs))
	})
}

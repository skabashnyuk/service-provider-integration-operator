package tokenstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	api "github.com/redhat-appstudio/service-provider-integration-operator/api/v1beta1"
)

type secretManagerTokenStorage struct {
	client *secretsmanager.Client
}

var _ TokenStorage = (*secretManagerTokenStorage)(nil)

// NewSecretManagerTokenStorage creates a new `TokenStorage` instance using ....
func NewSecretManagerTokenStorage(config aws.Config) (TokenStorage, error) {
	return &secretManagerTokenStorage{client: secretsmanager.NewFromConfig(config)}, nil
}

func (s *secretManagerTokenStorage) Initialize(ctx context.Context) error {
	return nil
}

func (s *secretManagerTokenStorage) Store(ctx context.Context, owner *api.SPIAccessToken, token *api.Token) error {
	secretName := fmt.Sprintf(vaultDataPathFormat, "spi", owner.Namespace, owner.Name)
	tokenJson, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("error marshalling the state: %w", err)
	}

	input := &secretsmanager.CreateSecretInput{
		Name:         aws.String(secretName),
		SecretBinary: tokenJson,
	}
	_, err = s.client.CreateSecret(ctx, input)
	if err != nil {
		return fmt.Errorf("error saving secret: %w", err)
	}
	return nil

}

func (s *secretManagerTokenStorage) Get(ctx context.Context, owner *api.SPIAccessToken) (*api.Token, error) {
	secretName := fmt.Sprintf(vaultDataPathFormat, "spi", owner.Namespace, owner.Name)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := s.client.GetSecretValue(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("not able to get secret from storage: %w", err)
	}

	token := api.Token{}
	if err := json.Unmarshal(result.SecretBinary, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token data: %w", err)
	}
	return &token, nil
}

func (s *secretManagerTokenStorage) Delete(ctx context.Context, owner *api.SPIAccessToken) error {
	secretName := fmt.Sprintf(vaultDataPathFormat, "spi", owner.Namespace, owner.Name)

	input := &secretsmanager.DeleteSecretInput{
		SecretId: aws.String(secretName),
	}
	_, err := s.client.DeleteSecret(ctx, input)
	if err != nil {
		return fmt.Errorf("error saving secret: %w", err)
	}
	return nil
}

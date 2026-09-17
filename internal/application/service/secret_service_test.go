package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/service"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// --- Mock Encryptor ---

type mockEncryptor struct {
	mock.Mock
}

func (m *mockEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	args := m.Called(plaintext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	args := m.Called(ciphertext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// --- Mock SecretRepository ---

type mockSecretRepo struct {
	mock.Mock
}

func (m *mockSecretRepo) Save(ctx context.Context, secret *model.Secret) error {
	args := m.Called(ctx, secret)
	return args.Error(0)
}

func (m *mockSecretRepo) FindByID(ctx context.Context, id string) (*model.Secret, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Secret), args.Error(1)
}

func (m *mockSecretRepo) FindBySuiteIDAndKey(ctx context.Context, suiteID, key string) (*model.Secret, error) {
	args := m.Called(ctx, suiteID, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Secret), args.Error(1)
}

func (m *mockSecretRepo) ListBySuiteID(ctx context.Context, suiteID string) ([]*model.Secret, error) {
	args := m.Called(ctx, suiteID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Secret), args.Error(1)
}

func (m *mockSecretRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// --- Mock SuiteRepository ---

type mockSuiteRepoForSecrets struct {
	mock.Mock
}

func (m *mockSuiteRepoForSecrets) Save(ctx context.Context, suite *model.TestSuite) error {
	args := m.Called(ctx, suite)
	return args.Error(0)
}

func (m *mockSuiteRepoForSecrets) FindByID(ctx context.Context, id string) (*model.TestSuite, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TestSuite), args.Error(1)
}

func (m *mockSuiteRepoForSecrets) FindByName(ctx context.Context, name string) (*model.TestSuite, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.TestSuite), args.Error(1)
}

func (m *mockSuiteRepoForSecrets) List(ctx context.Context) ([]*model.TestSuite, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.TestSuite), args.Error(1)
}

func (m *mockSuiteRepoForSecrets) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// --- Tests ---

func TestSecretService_CreateSecret(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully creates a secret", func(t *testing.T) {
		enc := new(mockEncryptor)
		repo := new(mockSecretRepo)
		suiteRepo := new(mockSuiteRepoForSecrets)

		suite, _ := model.NewTestSuite("test-suite", "desc")
		suiteRepo.On("FindByID", ctx, suite.ID()).Return(suite, nil)
		enc.On("Encrypt", []byte("my-password")).Return([]byte("encrypted-data"), nil)
		repo.On("Save", ctx, mock.AnythingOfType("*model.Secret")).Return(nil)

		svc := service.NewSecretService(suiteRepo, repo, enc)
		secret, err := svc.CreateSecret(ctx, inbound.CreateSecretCommand{
			SuiteID: suite.ID(),
			Key:     "DB_PASSWORD",
			Value:   "my-password",
		})

		require.NoError(t, err)
		require.NotNil(t, secret)
		assert.Equal(t, "DB_PASSWORD", secret.Key())
		assert.Equal(t, suite.ID(), secret.SuiteID())
		enc.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("fails on invalid secret key", func(t *testing.T) {
		enc := new(mockEncryptor)
		repo := new(mockSecretRepo)
		suiteRepo := new(mockSuiteRepoForSecrets)

		suite, _ := model.NewTestSuite("test-suite", "desc")
		suiteRepo.On("FindByID", ctx, suite.ID()).Return(suite, nil)

		svc := service.NewSecretService(suiteRepo, repo, enc)
		_, err := svc.CreateSecret(ctx, inbound.CreateSecretCommand{
			SuiteID: suite.ID(),
			Key:     "invalid-key",
			Value:   "value",
		})

		assert.ErrorIs(t, err, model.ErrInvalidSecretKey)
	})

	t.Run("fails on empty value", func(t *testing.T) {
		enc := new(mockEncryptor)
		repo := new(mockSecretRepo)
		suiteRepo := new(mockSuiteRepoForSecrets)

		suite, _ := model.NewTestSuite("test-suite", "desc")
		suiteRepo.On("FindByID", ctx, suite.ID()).Return(suite, nil)

		svc := service.NewSecretService(suiteRepo, repo, enc)
		_, err := svc.CreateSecret(ctx, inbound.CreateSecretCommand{
			SuiteID: suite.ID(),
			Key:     "DB_PASSWORD",
			Value:   "",
		})

		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("fails when suite not found", func(t *testing.T) {
		enc := new(mockEncryptor)
		repo := new(mockSecretRepo)
		suiteRepo := new(mockSuiteRepoForSecrets)

		suiteRepo.On("FindByID", ctx, "missing-suite").Return(nil, model.ErrNotFound)

		svc := service.NewSecretService(suiteRepo, repo, enc)
		_, err := svc.CreateSecret(ctx, inbound.CreateSecretCommand{
			SuiteID: "missing-suite",
			Key:     "KEY",
			Value:   "val",
		})

		assert.ErrorIs(t, err, model.ErrNotFound)
	})
}

func TestSecretService_ListSecrets(t *testing.T) {
	ctx := context.Background()

	t.Run("returns secrets for suite", func(t *testing.T) {
		enc := new(mockEncryptor)
		repo := new(mockSecretRepo)
		suiteRepo := new(mockSuiteRepoForSecrets)

		suite, _ := model.NewTestSuite("test-suite", "desc")
		suiteRepo.On("FindByID", ctx, suite.ID()).Return(suite, nil)

		s1, _ := model.NewSecret(suite.ID(), "KEY_A", []byte("enc1"))
		s2, _ := model.NewSecret(suite.ID(), "KEY_B", []byte("enc2"))
		repo.On("ListBySuiteID", ctx, suite.ID()).Return([]*model.Secret{s1, s2}, nil)

		svc := service.NewSecretService(suiteRepo, repo, enc)
		secrets, err := svc.ListSecrets(ctx, suite.ID())

		require.NoError(t, err)
		assert.Len(t, secrets, 2)
	})
}

func TestSecretService_UpdateSecret(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully updates secret value", func(t *testing.T) {
		enc := new(mockEncryptor)
		repo := new(mockSecretRepo)
		suiteRepo := new(mockSuiteRepoForSecrets)

		suite, _ := model.NewTestSuite("test-suite", "desc")
		suiteRepo.On("FindByID", ctx, suite.ID()).Return(suite, nil)

		existing, _ := model.NewSecret(suite.ID(), "DB_PASSWORD", []byte("old-enc"))
		repo.On("FindByID", ctx, existing.ID()).Return(existing, nil)
		enc.On("Encrypt", []byte("new-password")).Return([]byte("new-encrypted"), nil)
		repo.On("Save", ctx, mock.AnythingOfType("*model.Secret")).Return(nil)

		svc := service.NewSecretService(suiteRepo, repo, enc)
		updated, err := svc.UpdateSecret(ctx, inbound.UpdateSecretCommand{
			SuiteID:  suite.ID(),
			SecretID: existing.ID(),
			Value:    "new-password",
		})

		require.NoError(t, err)
		assert.Equal(t, []byte("new-encrypted"), updated.EncryptedValue())
	})

	t.Run("fails when secret not in suite", func(t *testing.T) {
		enc := new(mockEncryptor)
		repo := new(mockSecretRepo)
		suiteRepo := new(mockSuiteRepoForSecrets)

		suite, _ := model.NewTestSuite("test-suite", "desc")
		suiteRepo.On("FindByID", ctx, suite.ID()).Return(suite, nil)

		otherSuiteSecret, _ := model.NewSecret("other-suite", "KEY", []byte("enc"))
		repo.On("FindByID", ctx, otherSuiteSecret.ID()).Return(otherSuiteSecret, nil)

		svc := service.NewSecretService(suiteRepo, repo, enc)
		_, err := svc.UpdateSecret(ctx, inbound.UpdateSecretCommand{
			SuiteID:  suite.ID(),
			SecretID: otherSuiteSecret.ID(),
			Value:    "new-value",
		})

		assert.ErrorIs(t, err, model.ErrNotFound)
	})
}

func TestSecretService_DeleteSecret(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully deletes secret", func(t *testing.T) {
		enc := new(mockEncryptor)
		repo := new(mockSecretRepo)
		suiteRepo := new(mockSuiteRepoForSecrets)

		suite, _ := model.NewTestSuite("test-suite", "desc")
		suiteRepo.On("FindByID", ctx, suite.ID()).Return(suite, nil)

		existing, _ := model.NewSecret(suite.ID(), "DB_PASSWORD", []byte("enc"))
		repo.On("FindByID", ctx, existing.ID()).Return(existing, nil)
		repo.On("Delete", ctx, existing.ID()).Return(nil)

		svc := service.NewSecretService(suiteRepo, repo, enc)
		err := svc.DeleteSecret(ctx, suite.ID(), existing.ID())

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestSecretService_GetDecryptedSecrets(t *testing.T) {
	ctx := context.Background()

	t.Run("decrypts requested secret keys", func(t *testing.T) {
		enc := new(mockEncryptor)
		repo := new(mockSecretRepo)
		suiteRepo := new(mockSuiteRepoForSecrets)

		suiteID := "suite-1"
		s1, _ := model.NewSecret(suiteID, "DB_PASSWORD", []byte("enc-pw"))
		s2, _ := model.NewSecret(suiteID, "API_KEY", []byte("enc-key"))

		repo.On("ListBySuiteID", ctx, suiteID).Return([]*model.Secret{s1, s2}, nil)
		enc.On("Decrypt", []byte("enc-pw")).Return([]byte("my-password"), nil)
		enc.On("Decrypt", []byte("enc-key")).Return([]byte("my-api-key"), nil)

		svc := service.NewSecretService(suiteRepo, repo, enc)
		decrypted, err := svc.GetDecryptedSecrets(ctx, suiteID, []string{"DB_PASSWORD", "API_KEY"})

		require.NoError(t, err)
		assert.Equal(t, "my-password", decrypted["DB_PASSWORD"])
		assert.Equal(t, "my-api-key", decrypted["API_KEY"])
	})

	t.Run("fails when a requested key is missing", func(t *testing.T) {
		enc := new(mockEncryptor)
		repo := new(mockSecretRepo)
		suiteRepo := new(mockSuiteRepoForSecrets)

		suiteID := "suite-1"
		s1, _ := model.NewSecret(suiteID, "DB_PASSWORD", []byte("enc-pw"))

		repo.On("ListBySuiteID", ctx, suiteID).Return([]*model.Secret{s1}, nil)

		svc := service.NewSecretService(suiteRepo, repo, enc)
		_, err := svc.GetDecryptedSecrets(ctx, suiteID, []string{"DB_PASSWORD", "MISSING_KEY"})

		assert.ErrorIs(t, err, model.ErrMissingSecret)
	})
}

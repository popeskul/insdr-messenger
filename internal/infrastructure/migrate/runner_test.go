package migrate_test

import (
	"errors"
	"testing"

	"github.com/popeskul/insdr-messenger/internal/infrastructure/migrate"
)

// MockRunner implements a test version of migration runner
type MockRunner struct {
	version uint
	dirty   bool
	runErr  error
}

func (m *MockRunner) Run() error {
	if m.runErr != nil {
		return m.runErr
	}
	m.version = 1
	return nil
}

func (m *MockRunner) Rollback() error {
	m.version = 0
	return nil
}

func (m *MockRunner) Version() (uint, bool, error) {
	return m.version, m.dirty, nil
}

func TestMigrations(t *testing.T) {
	tests := []struct {
		name         string
		runner       *MockRunner
		operation    func(*MockRunner) error
		checkVersion func(uint, bool, error) error
		wantErr      bool
	}{
		{
			name:   "Run migrations successfully",
			runner: &MockRunner{},
			operation: func(r *MockRunner) error {
				return r.Run()
			},
			checkVersion: func(version uint, dirty bool, err error) error {
				if err != nil {
					return err
				}
				if dirty {
					return errors.New("database is in dirty state after migration")
				}
				if version == 0 {
					return errors.New("expected version to be greater than 0")
				}
				return nil
			},
			wantErr: false,
		},
		{
			name:   "Run migrations with error",
			runner: &MockRunner{runErr: errors.New("migration failed")},
			operation: func(r *MockRunner) error {
				return r.Run()
			},
			checkVersion: func(version uint, dirty bool, err error) error {
				// Version should remain 0 after failed migration
				if version != 0 {
					return errors.New("expected version to be 0 after failed migration")
				}
				return nil
			},
			wantErr: true,
		},
		{
			name:   "Rollback migration successfully",
			runner: &MockRunner{version: 1},
			operation: func(r *MockRunner) error {
				return r.Rollback()
			},
			checkVersion: func(version uint, dirty bool, err error) error {
				if err != nil {
					return err
				}
				if version != 0 {
					return errors.New("expected version to be 0 after rollback")
				}
				return nil
			},
			wantErr: false,
		},
		{
			name:   "Check version with dirty state",
			runner: &MockRunner{version: 2, dirty: true},
			operation: func(r *MockRunner) error {
				// No operation, just check version
				return nil
			},
			checkVersion: func(version uint, dirty bool, err error) error {
				if !dirty {
					return errors.New("expected database to be in dirty state")
				}
				if version != 2 {
					return errors.New("expected version to be 2")
				}
				return nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation(tt.runner)
			if (err != nil) != tt.wantErr {
				t.Errorf("operation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			version, dirty, err := tt.runner.Version()
			if checkErr := tt.checkVersion(version, dirty, err); checkErr != nil {
				t.Errorf("version check failed: %v", checkErr)
			}
		})
	}

	// Test actual runner creation
	t.Run("Create runner with config", func(t *testing.T) {
		config := &migrate.Config{
			DatabaseURL:    "postgres://test:test@localhost/test",
			MigrationsPath: "../../../migrations",
		}

		runner := migrate.NewRunner(config)
		if runner == nil {
			t.Fatal("Expected runner to be created")
		}
	})
}

package providers

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockProviderFactory_DefaultsReturnMockPackageManager(t *testing.T) {
	var f MockProviderFactory

	pm := f.CreateNPMProvider()
	assert.NotNil(t, pm)
	_, ok := pm.(*MockPackageManager)
	assert.True(t, ok)

	pm = f.CreatePyPIProvider()
	assert.NotNil(t, pm)
	_, ok = pm.(*MockPackageManager)
	assert.True(t, ok)

	pm = f.CreateGolangProvider()
	assert.NotNil(t, pm)
	_, ok = pm.(*MockPackageManager)
	assert.True(t, ok)

	pm = f.CreateCargoProvider()
	assert.NotNil(t, pm)
	_, ok = pm.(*MockPackageManager)
	assert.True(t, ok)
}

func TestMockProviderFactory_UsesInjectedManagers(t *testing.T) {
	custom := &MockPackageManager{}
	f := MockProviderFactory{
		MockNPMProvider:    custom,
		MockPyPIProvider:   custom,
		MockGolangProvider: custom,
		MockCargoProvider:  custom,
	}

	assert.Same(t, custom, f.CreateNPMProvider())
	assert.Same(t, custom, f.CreatePyPIProvider())
	assert.Same(t, custom, f.CreateGolangProvider())
	assert.Same(t, custom, f.CreateCargoProvider())
}

func TestMockPackageManager_DefaultReturns(t *testing.T) {
	m := &MockPackageManager{}
	assert.False(t, m.Install("pkg:any/x", "1.0.0"))
	assert.False(t, m.Remove("pkg:any/x"))
	assert.False(t, m.Update("pkg:any/x"))
}

func TestMockPackageManager_GetLatestVersionFuncBranch(t *testing.T) {
	called := 0
	m := &MockPackageManager{
		GetLatestVersionFunc: func(packageName string) (string, error) {
			called++
			if packageName != "foo" {
				return "", errors.New("unexpected package")
			}
			return "9.9.9", nil
		},
	}

	ver, err := m.getLatestVersion("foo")
	assert.NoError(t, err)
	assert.Equal(t, "9.9.9", ver)
	assert.Equal(t, 1, called)
}

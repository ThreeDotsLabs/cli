package files

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGoVendorDir(t *testing.T) {
	fs := afero.NewMemMapFs()

	require.NoError(t, afero.WriteFile(fs, "/exercise/go.mod", []byte("module foo\n"), 0644))
	require.NoError(t, fs.MkdirAll("/exercise/vendor/example.com/dep", 0755))

	require.NoError(t, afero.WriteFile(fs, "/exercise/service-a/go.mod", []byte("module a\n"), 0644))
	require.NoError(t, fs.MkdirAll("/exercise/service-a/vendor", 0755))

	// A domain package that happens to be named "vendor" — no go.mod sibling.
	require.NoError(t, fs.MkdirAll("/exercise/internal/marketplace/vendor", 0755))

	require.NoError(t, fs.MkdirAll("/exercise/internal/orders", 0755))

	testCases := []struct {
		name     string
		dirPath  string
		expected bool
	}{
		{
			name:     "vendor_at_module_root",
			dirPath:  "/exercise/vendor",
			expected: true,
		},
		{
			name:     "vendor_at_nested_module_root",
			dirPath:  "/exercise/service-a/vendor",
			expected: true,
		},
		{
			name:     "user_package_named_vendor",
			dirPath:  "/exercise/internal/marketplace/vendor",
			expected: false,
		},
		{
			name:     "not_named_vendor",
			dirPath:  "/exercise/internal/orders",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isGoVendorDir(fs, tc.dirPath))
		})
	}
}

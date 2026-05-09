package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestEmit_Types_FixtureGolden(t *testing.T) {
	api, err := loadAPI("../../testdata/golden/api_small_fixture.json")
	require.NoError(t, err)

	tmp := t.TempDir()
	e := newEmitter(api, tmp)
	require.NoError(t, e.emitTypes())

	got, err := os.ReadFile(filepath.Join(tmp, "types.gen.go"))
	require.NoError(t, err)

	goldenPath := "../../testdata/golden/types.gen.go"
	if *updateGolden {
		require.NoError(t, os.WriteFile(goldenPath, got, 0o600))
		return
	}
	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "missing golden; run `go test -update ./cmd/genapi/...`")
	require.Equal(t, string(expected), string(got))
}

func TestEmit_Enums_FixtureGolden(t *testing.T) {
	api, err := loadAPI("../../testdata/golden/api_small_fixture.json")
	require.NoError(t, err)

	tmp := t.TempDir()
	e := newEmitter(api, tmp)
	require.NoError(t, e.emitEnums())

	got, err := os.ReadFile(filepath.Join(tmp, "enums.gen.go"))
	require.NoError(t, err)

	goldenPath := "../../testdata/golden/enums.gen.go"
	if *updateGolden {
		require.NoError(t, os.WriteFile(goldenPath, got, 0o600))
		return
	}
	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "missing golden; run `go test -update ./cmd/genapi/...`")
	require.Equal(t, string(expected), string(got))
}

func TestEmit_Methods_FixtureGolden(t *testing.T) {
	api, err := loadAPI("../../testdata/golden/api_small_fixture.json")
	require.NoError(t, err)

	tmp := t.TempDir()
	e := newEmitter(api, tmp)
	require.NoError(t, e.emitTypes()) // some methods reference types
	require.NoError(t, e.emitMethods())

	got, err := os.ReadFile(filepath.Join(tmp, "methods.gen.go"))
	require.NoError(t, err)

	goldenPath := "../../testdata/golden/methods.gen.go"
	if *updateGolden {
		require.NoError(t, os.WriteFile(goldenPath, got, 0o600))
		return
	}
	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "missing golden; run `go test -update ./cmd/genapi/...`")
	require.Equal(t, string(expected), string(got))
}

func TestEmit_Tests_FixtureGolden(t *testing.T) {
	api, err := loadAPI("../../testdata/golden/api_small_fixture.json")
	require.NoError(t, err)

	tmp := t.TempDir()
	e := newEmitter(api, tmp)
	require.NoError(t, e.emitTests())

	got, err := os.ReadFile(filepath.Join(tmp, "methods_gen_test.go"))
	require.NoError(t, err)

	goldenPath := "../../testdata/golden/methods_gen_test.go"
	if *updateGolden {
		require.NoError(t, os.WriteFile(goldenPath, got, 0o600))
		return
	}
	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "missing golden; run `go test -update ./cmd/genapi/...`")
	require.Equal(t, string(expected), string(got))
}

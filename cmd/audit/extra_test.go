package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// loadIR
// ---------------------------------------------------------------------------

func TestLoadIR_ValidFile(t *testing.T) {
	api := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "getMe", Returns: spec.TypeRef{Kind: spec.KindNamed, Name: "User"}},
		},
	}
	data, err := json.Marshal(api)
	require.NoError(t, err)

	tmp := filepath.Join(t.TempDir(), "api.json")
	require.NoError(t, os.WriteFile(tmp, data, 0o600))

	loaded, err := loadIR(tmp)
	require.NoError(t, err)
	require.Len(t, loaded.Methods, 1)
	require.Equal(t, "getMe", loaded.Methods[0].Name)
}

func TestLoadIR_MissingFile(t *testing.T) {
	_, err := loadIR("/nonexistent/path/api.json")
	require.Error(t, err)
	require.Contains(t, err.Error(), "open IR")
}

func TestLoadIR_InvalidJSON(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(tmp, []byte("not json"), 0o600))

	_, err := loadIR(tmp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode IR")
}

// ---------------------------------------------------------------------------
// auditBool
// ---------------------------------------------------------------------------

func TestAuditBool_LongDocTruncated(t *testing.T) {
	longDoc := make([]byte, 200)
	for i := range longDoc {
		longDoc[i] = 'a'
	}
	api := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "myMethod", Doc: string(longDoc), Returns: spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}},
		},
	}
	problems := auditBool(api, &spec.Overrides{})
	require.Len(t, problems, 1)
	require.Contains(t, problems[0], "…")
}

func TestAuditBool_TrueIsReturnedVariant(t *testing.T) {
	api := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "doThing", Doc: "true is returned on success.", Returns: spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}},
		},
	}
	require.Empty(t, auditBool(api, &spec.Overrides{}))
}

func TestAuditBool_ReturnsBoolean(t *testing.T) {
	api := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "doThing", Doc: "Returns Boolean on success.", Returns: spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}},
		},
	}
	require.Empty(t, auditBool(api, &spec.Overrides{}))
}

// ---------------------------------------------------------------------------
// formatTypeRef
// ---------------------------------------------------------------------------

func TestFormatTypeRef_AllBranches(t *testing.T) {
	cases := []struct {
		tr   spec.TypeRef
		want string
	}{
		{spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}, "bool"},
		{spec.TypeRef{Kind: spec.KindNamed, Name: "User"}, "User"},
		{spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindNamed, Name: "Update"}}, "[]Update"},
		{spec.TypeRef{Kind: spec.KindArray}, "[]any"},
		{spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"int64", "string"}}, "(int64 | string)"},
		{spec.TypeRef{Kind: spec.Kind(99)}, "?"},
	}
	for _, c := range cases {
		got := formatTypeRef(c.tr)
		require.Equal(t, c.want, got, "for kind=%v name=%v", c.tr.Kind, c.tr.Name)
	}
}

// ---------------------------------------------------------------------------
// auditDrift
// ---------------------------------------------------------------------------

func TestAuditDrift_InvalidRefReturnsError(t *testing.T) {
	cur := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "getMe", Returns: spec.TypeRef{Kind: spec.KindNamed, Name: "User"}},
		},
	}
	_, err := auditDrift("internal/spec/api.json", "THIS_REF_DOES_NOT_EXIST", cur)
	require.Error(t, err)
}

func TestAuditDrift_SameRefNoDrift(t *testing.T) {
	irPath := "../../internal/spec/api.json"
	cur, err := loadIR(irPath)
	if err != nil {
		t.Skip("api.json not available, skipping drift test")
	}
	changes, err := auditDrift(irPath, "HEAD", cur)
	require.NoError(t, err)
	require.Empty(t, changes)
}

func TestAuditDrift_InvalidJSONFromGit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script not supported on Windows")
	}
	tmp := t.TempDir()
	fakeGit := filepath.Join(tmp, "git")
	require.NoError(t, os.WriteFile(fakeGit, []byte("#!/bin/sh\necho 'not valid json'\n"), 0o600))
	require.NoError(t, os.Chmod(fakeGit, 0o755))

	origPATH := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", origPATH) })
	_ = os.Setenv("PATH", tmp+string(os.PathListSeparator)+origPATH)

	_, err := auditDrift("internal/spec/api.json", "HEAD", &spec.API{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode")
}

// ---------------------------------------------------------------------------
// auditAny
// ---------------------------------------------------------------------------

func TestAuditAny_FlagsUnknownMethodReturn(t *testing.T) {
	api := &spec.API{
		Methods: []spec.MethodDecl{
			{
				Name:    "weirdMethod",
				Returns: spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"A", "B", "C"}},
			},
		},
	}
	out := auditAny(api)
	require.Len(t, out, 1)
	require.Contains(t, out[0], "any return: weirdMethod")
}

func TestAuditAny_FlagsUnknownMethodParam(t *testing.T) {
	api := &spec.API{
		Methods: []spec.MethodDecl{
			{
				Name: "weirdMethod",
				Params: []spec.Field{
					{Name: "Thing", JSONName: "thing", Type: spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"X", "Y", "Z"}}},
				},
			},
		},
	}
	out := auditAny(api)
	require.Len(t, out, 1)
	require.Contains(t, out[0], "any param: weirdMethod.Thing")
}

// ---------------------------------------------------------------------------
// diffSignatures
// ---------------------------------------------------------------------------

func TestDiffSignatures_UnchangedNoDrift(t *testing.T) {
	prev := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "getMe", Returns: spec.TypeRef{Kind: spec.KindNamed, Name: "User"}},
		},
	}
	cur := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "getMe", Returns: spec.TypeRef{Kind: spec.KindNamed, Name: "User"}},
		},
	}
	require.Empty(t, diffSignatures(prev, cur))
}

// ---------------------------------------------------------------------------
// typeRefEqual
// ---------------------------------------------------------------------------

func TestTypeRefEqual_ArrayNilElemDiffers(t *testing.T) {
	a := spec.TypeRef{Kind: spec.KindArray}
	b := spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindNamed, Name: "Update"}}
	require.False(t, typeRefEqual(a, b))
	require.False(t, typeRefEqual(b, a))
}

// ---------------------------------------------------------------------------
// TestHelperMain: subprocess helper for main() coverage.
//
// When AUDIT_HELPER_MAIN=1 is set, this function:
//   1. Resets flag.CommandLine so main()'s flag.Parse() gets a clean slate.
//   2. Sets os.Args to the args encoded in AUDIT_HELPER_ARGS.
//   3. Calls main() which calls os.Exit — the test process terminates with
//      main's exit code, which the parent test captures.
// ---------------------------------------------------------------------------

func TestHelperMain(t *testing.T) {
	if os.Getenv("AUDIT_HELPER_MAIN") != "1" {
		t.Skip("not running as subprocess helper")
	}
	// Reset flag.CommandLine so main()'s flag.Parse() gets a clean slate.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Decode args from JSON-encoded env var.
	encoded := os.Getenv("AUDIT_HELPER_ARGS")
	if encoded != "" {
		var args []string
		if err := json.Unmarshal([]byte(encoded), &args); err == nil {
			os.Args = append([]string{os.Args[0]}, args...)
		}
	} else {
		os.Args = os.Args[:1]
	}

	main()
}

// runMain runs main() via the test binary subprocess so that coverage counters
// from main() are included in the profile. Args are JSON-encoded in an env var
// to avoid conflicts with the test binary's own flag parsing.
func runMain(t *testing.T, extraEnv []string, args ...string) (string, int) {
	t.Helper()
	argsJSON, _ := json.Marshal(args)
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperMain", "-test.v=false")
	cmd.Env = append(os.Environ(), "AUDIT_HELPER_MAIN=1", "AUDIT_HELPER_ARGS="+string(argsJSON))
	cmd.Env = append(cmd.Env, extraEnv...)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
	}
	return string(out), code
}

// ---------------------------------------------------------------------------
// main() integration tests — exercise main() code paths via subprocess
// ---------------------------------------------------------------------------

func TestMain_CleanExitsZero(t *testing.T) {
	tmp := t.TempDir()
	ir := writeIR(t, tmp, &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "getMe", Doc: "Returns True on success.", Returns: spec.TypeRef{Kind: spec.KindNamed, Name: "User"}},
		},
	})
	ov := writeOverrides(t, tmp)

	out, code := runMain(t, nil, "-ir", ir, "-overrides", ov)
	require.Equal(t, exitClean, code, "expected exit 0 (clean)\nout: %s", out)
	require.Contains(t, out, "clean")
}

func TestMain_FallbackExitsOne(t *testing.T) {
	tmp := t.TempDir()
	ir := writeIR(t, tmp, &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "doSomething", Doc: "Does something.", Returns: spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}},
		},
	})
	ov := writeOverrides(t, tmp)

	out, code := runMain(t, nil, "-ir", ir, "-overrides", ov)
	require.Equal(t, exitFallback, code, "expected exit 1 (fallback)\nout: %s", out)
	require.Contains(t, out, "bool fallback")
}

func TestMain_InvalidIRExitsThree(t *testing.T) {
	tmp := t.TempDir()
	bad := filepath.Join(tmp, "bad.json")
	require.NoError(t, os.WriteFile(bad, []byte("not json"), 0o600))
	ov := writeOverrides(t, tmp)

	out, code := runMain(t, nil, "-ir", bad, "-overrides", ov)
	require.Equal(t, exitInvalid, code, "expected exit 3 (invalid IR)\nout: %s", out)
}

func TestMain_InvalidOverridesExitsThree(t *testing.T) {
	tmp := t.TempDir()
	ir := writeIR(t, tmp, &spec.API{})
	bad := filepath.Join(tmp, "bad_ov.json")
	require.NoError(t, os.WriteFile(bad, []byte("not json"), 0o600))

	out, code := runMain(t, nil, "-ir", ir, "-overrides", bad)
	require.Equal(t, exitInvalid, code, "expected exit 3 (invalid overrides)\nout: %s", out)
}

func TestMain_DriftDetectedExitsTwo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script not supported on Windows")
	}
	tmp := t.TempDir()

	prevAPI := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "getMe", Returns: spec.TypeRef{Kind: spec.KindNamed, Name: "User"}},
		},
	}
	curAPI := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "getMe", Returns: spec.TypeRef{Kind: spec.KindNamed, Name: "Message"}},
		},
	}

	curIR := writeIR(t, tmp, curAPI)
	ov := writeOverrides(t, tmp)

	prevData, err := json.Marshal(prevAPI)
	require.NoError(t, err)
	prevFile := filepath.Join(tmp, "prev.json")
	require.NoError(t, os.WriteFile(prevFile, prevData, 0o600))

	fakeGit := filepath.Join(tmp, "git")
	script := fmt.Sprintf("#!/bin/sh\ncat %s\n", prevFile)
	require.NoError(t, os.WriteFile(fakeGit, []byte(script), 0o600))
	require.NoError(t, os.Chmod(fakeGit, 0o755))

	newPATH := tmp + string(os.PathListSeparator) + os.Getenv("PATH")

	out, code := runMain(t,
		[]string{"PATH=" + newPATH},
		"-ir", curIR, "-overrides", ov, "-drift", "-against", "HEAD~1",
	)
	require.Equal(t, exitDrift, code, "expected exit 2 (drift)\nout: %s", out)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func writeIR(t *testing.T, dir string, api *spec.API) string {
	t.Helper()
	data, err := json.Marshal(api)
	require.NoError(t, err)
	p := filepath.Join(dir, "api.json")
	require.NoError(t, os.WriteFile(p, data, 0o600))
	return p
}

func writeOverrides(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "overrides.json")
	require.NoError(t, os.WriteFile(p, []byte("{}"), 0o600))
	return p
}

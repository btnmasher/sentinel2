package buildmeta

import "testing"

func TestDeriveBuildVersion_UsesExplicitEnv(t *testing.T) {
	t.Setenv("BUILD_VERSION", "v9.9.9-test")
	got, err := DeriveBuildVersion()
	if err != nil {
		t.Fatalf("DeriveBuildVersion() error = %v", err)
	}
	if got != "v9.9.9-test" {
		t.Fatalf("DeriveBuildVersion() = %q, want %q", got, "v9.9.9-test")
	}
}

func TestDeriveBuildVersion_NoGitRepoReturnsEmpty(t *testing.T) {
	t.Setenv("BUILD_VERSION", "")
	t.Setenv("PATH", "")
	got, err := DeriveBuildVersion()
	if err != nil {
		t.Fatalf("DeriveBuildVersion() error = %v", err)
	}
	if got != "" {
		t.Fatalf("DeriveBuildVersion() = %q, want empty", got)
	}
}

func TestSafeBranchName(t *testing.T) {
	got := safeBranchName(" feature/foo bar@baz ")
	want := "feature-foo-bar-baz"
	if got != want {
		t.Fatalf("safeBranchName() = %q, want %q", got, want)
	}
}

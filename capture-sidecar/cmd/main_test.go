package main

import "testing"

// TestEnvOrDefault_UsesEnvWhenSet is the regression test for F-188: the
// TLS_CERT_FILE/TLS_KEY_FILE/TLS_CA_FILE env vars specs.md documents as
// required (and that the operator's buildCaptureEphemeralContainer actually
// sets) previously had no effect at all - the sidecar only appeared to
// honor them because its flag defaults happened to equal the same paths.
func TestEnvOrDefault_UsesEnvWhenSet(t *testing.T) {
	const name = "GAMEPLANE_CAPTURE_SIDECAR_TEST_ENV_OR_DEFAULT"

	t.Run("unset falls back", func(t *testing.T) {
		t.Setenv(name, "")
		if got := envOrDefault(name, "/fallback/path"); got != "/fallback/path" {
			t.Errorf("envOrDefault = %q, want fallback %q", got, "/fallback/path")
		}
	})

	t.Run("set value takes effect", func(t *testing.T) {
		t.Setenv(name, "/mounted/custom.crt")
		if got := envOrDefault(name, "/fallback/path"); got != "/mounted/custom.crt" {
			t.Errorf("envOrDefault = %q, want the env value %q", got, "/mounted/custom.crt")
		}
	})
}

// TestDefaultVolumeBudgetBytes_MatchesOperator pins defaultVolumeBudgetBytes
// to the value it must keep matching by hand: the operator's
// captureVolumeBudgetBytes in
// operator/internal/controller/gameserver_controller.go (1Gi "captures"
// emptyDir SizeLimit minus a 10% safety margin, i.e. 966367642 bytes). The
// two constants can't share Go code across modules, so this test exists to
// catch an accidental edit to either side going unnoticed (see
// defaultVolumeBudgetBytes' doc comment in main.go).
func TestDefaultVolumeBudgetBytes_MatchesOperator(t *testing.T) {
	const wantOperatorCaptureVolumeBudgetBytes = 966367642
	if defaultVolumeBudgetBytes != wantOperatorCaptureVolumeBudgetBytes {
		t.Errorf("defaultVolumeBudgetBytes = %d, want %d (must match captureVolumeBudgetBytes in operator/internal/controller/gameserver_controller.go)",
			defaultVolumeBudgetBytes, wantOperatorCaptureVolumeBudgetBytes)
	}
}

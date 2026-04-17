package redact

import (
	"strings"
	"testing"
)

func TestRedact_AWSAccessKeyID(t *testing.T) {
	in := "credentials: AKIAIOSFODNN7EXAMPLE used for S3"
	r := Redact(in)
	if r.Counts["aws-access-key-id"] != 1 {
		t.Fatalf("expected 1 aws key redaction, got %d", r.Counts["aws-access-key-id"])
	}
	if strings.Contains(r.Text, "AKIAIOSFODNN7EXAMPLE") {
		t.Fatalf("aws key was not scrubbed: %q", r.Text)
	}
	if !strings.Contains(r.Text, "[REDACTED:aws-access-key-id]") {
		t.Fatalf("missing redaction marker: %q", r.Text)
	}
}

func TestRedact_AWSAccessKeyPrefixes(t *testing.T) {
	// All documented AWS key prefixes should match.
	prefixes := []string{"AKIA", "ASIA", "AIDA", "AGPA", "AROA", "AIPA", "ANPA", "ANVA", "ASCA"}
	for _, p := range prefixes {
		in := p + "0123456789ABCDEF"
		r := Redact(in)
		if r.Counts["aws-access-key-id"] != 1 {
			t.Errorf("prefix %q: expected redaction, got text=%q", p, r.Text)
		}
	}
}

func TestRedact_AWSAccessKeyID_FalsePositives(t *testing.T) {
	// 19-char and 21-char "AKIA..." values must not match.
	tests := []string{
		"AKIA0123456789ABCDE",     // 19 chars (too short)
		"AKIA0123456789ABCDEFG",   // 21 chars (too long; \b ensures not matched)
		"notAKIAIOSFODNN7EXAMPLE", // no word boundary
	}
	for _, in := range tests {
		r := Redact(in)
		if r.Counts["aws-access-key-id"] != 0 {
			t.Errorf("unexpected match for %q: %q", in, r.Text)
		}
	}
}

func TestRedact_GitHubTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"ghp", "ghp_" + strings.Repeat("a", 36), "github-token"},
		{"gho", "gho_" + strings.Repeat("b", 36), "github-token"},
		{"ghu", "ghu_" + strings.Repeat("c", 36), "github-token"},
		{"ghs", "ghs_" + strings.Repeat("d", 36), "github-token"},
		{"ghr", "ghr_" + strings.Repeat("e", 36), "github-token"},
		{"fine-grained", "github_pat_" + strings.Repeat("A", 82), "github-fine-grained-pat"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Redact("token=" + tt.input)
			if r.Counts[tt.want] != 1 {
				t.Errorf("want %s redaction, got counts=%v text=%q", tt.want, r.Counts, r.Text)
			}
			if strings.Contains(r.Text, tt.input) {
				t.Errorf("token leaked: %q", r.Text)
			}
		})
	}
}

func TestRedact_PrivateKeyPEM(t *testing.T) {
	in := `before
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAx...
fakebase64contenthere
-----END RSA PRIVATE KEY-----
after`
	r := Redact(in)
	if r.Counts["private-key-pem"] != 1 {
		t.Fatalf("expected pem redaction, got %v", r.Counts)
	}
	if strings.Contains(r.Text, "MIIEowIBAAKCAQEAx") {
		t.Fatalf("pem body leaked: %q", r.Text)
	}
	if !strings.Contains(r.Text, "before") || !strings.Contains(r.Text, "after") {
		t.Fatalf("surrounding context lost: %q", r.Text)
	}
}

func TestRedact_PrivateKeyPEM_Variants(t *testing.T) {
	variants := []string{
		"-----BEGIN PRIVATE KEY-----\nbody\n-----END PRIVATE KEY-----",
		"-----BEGIN EC PRIVATE KEY-----\nbody\n-----END EC PRIVATE KEY-----",
		"-----BEGIN OPENSSH PRIVATE KEY-----\nbody\n-----END OPENSSH PRIVATE KEY-----",
		"-----BEGIN DSA PRIVATE KEY-----\nbody\n-----END DSA PRIVATE KEY-----",
		"-----BEGIN PGP PRIVATE KEY BLOCK-----\nbody\n-----END PGP PRIVATE KEY BLOCK-----",
	}
	for _, v := range variants {
		r := Redact(v)
		if r.Counts["private-key-pem"] != 1 {
			t.Errorf("variant not redacted: %q -> %v", v, r.Counts)
		}
	}
}

func TestRedact_SlackGoogleStripe(t *testing.T) {
	tests := []struct {
		name  string
		input string
		key   string
	}{
		{"slack-bot", "xoxb-1234567890-abcdefghij", "slack-token"},
		{"slack-user", "xoxp-1234567890-abcdefghij", "slack-token"},
		{"google", "AIza" + strings.Repeat("a", 35), "google-api-key"},
		{"stripe-live", "sk_live_" + strings.Repeat("a", 24), "stripe-secret-key"},
		{"stripe-test", "sk_test_" + strings.Repeat("b", 24), "stripe-secret-key"},
		{"stripe-restricted", "rk_live_" + strings.Repeat("c", 24), "stripe-secret-key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Redact("value=" + tt.input)
			if r.Counts[tt.key] < 1 {
				t.Errorf("want %s redaction, got counts=%v text=%q", tt.key, r.Counts, r.Text)
			}
		})
	}
}

func TestRedact_OpenAIStyleKey(t *testing.T) {
	in := "sk-" + strings.Repeat("A1", 25)
	r := Redact(in)
	if r.Counts["openai-style-secret-key"] != 1 {
		t.Fatalf("expected openai-style redaction, got %v", r.Counts)
	}
}

func TestRedact_JWT(t *testing.T) {
	jwt := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	r := Redact("Authorization: Bearer " + jwt)
	if r.Counts["jwt"] != 1 {
		t.Fatalf("expected jwt redaction, got %v", r.Counts)
	}
	if strings.Contains(r.Text, jwt) {
		t.Fatalf("jwt leaked: %q", r.Text)
	}
}

func TestRedact_AssignmentSecret(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		redact bool
	}{
		{"api_key eq", `api_key = "aB3dEfGh1JkLmN2pQrStUv"`, true},
		{"password colon", "password: SuperSecret123Passw0rd!", true},
		{"token env", "GITHUB_TOKEN=Xy9ZaBcD3fGh4IjKlM5n", true},
		{"secret yaml", "secret: Sd8f7g9h2j3k4l5m6n7o8p", true},
		{"placeholder your", `api_key = "your-api-key-here"`, false},
		{"placeholder xxx", `api_key = "xxxxxxxxxxxxxxxxxxxx"`, false},
		{"short value", `api_key = "abc"`, false},
		{"low entropy", `token = "aaaaaaaaaaaaaaaaaaaaaaa"`, false},
		{"all lowercase prose", `password = "thisisnotasecretreallyword"`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Redact(tt.input)
			got := r.Counts["assignment-secret"] > 0
			if got != tt.redact {
				t.Errorf("input %q: redacted=%v want=%v (text=%q)", tt.input, got, tt.redact, r.Text)
			}
		})
	}
}

func TestRedact_AssignmentSecret_PreservesKey(t *testing.T) {
	in := `api_key = "aB3dEfGh1JkLmN2pQrStUv"`
	r := Redact(in)
	if !strings.Contains(r.Text, "api_key =") {
		t.Fatalf("key name dropped: %q", r.Text)
	}
	if !strings.Contains(r.Text, "[REDACTED:assignment-secret]") {
		t.Fatalf("missing marker: %q", r.Text)
	}
}

func TestRedact_MultipleSecrets(t *testing.T) {
	in := `AWS key: AKIAIOSFODNN7EXAMPLE
GH token: ghp_` + strings.Repeat("x", 36) + `
password = "Sd8f7g9h2j3k4l5m6n7o8p"`
	r := Redact(in)
	if r.Counts["aws-access-key-id"] != 1 {
		t.Errorf("aws key not redacted: %v", r.Counts)
	}
	if r.Counts["github-token"] != 1 {
		t.Errorf("github token not redacted: %v", r.Counts)
	}
	if r.Counts["assignment-secret"] != 1 {
		t.Errorf("assignment secret not redacted: %v", r.Counts)
	}
	if r.Total() != 3 {
		t.Errorf("expected total 3, got %d", r.Total())
	}
}

func TestRedact_NoSecrets(t *testing.T) {
	// Typical PR body with no secrets should pass through unchanged.
	in := `This PR fixes a bug in the loader.

See issue #42 for details. Commit sha is 3ccf1988a7c64cec7b9adb.`
	r := Redact(in)
	if r.Total() != 0 {
		t.Fatalf("false positive: %v text=%q", r.Counts, r.Text)
	}
	if r.Text != in {
		t.Fatalf("text modified: %q", r.Text)
	}
}

func TestRedact_Idempotent(t *testing.T) {
	in := "key: AKIAIOSFODNN7EXAMPLE"
	first := Redact(in)
	second := Redact(first.Text)
	if second.Total() != 0 {
		t.Fatalf("second pass redacted extra: %v text=%q", second.Counts, second.Text)
	}
	if second.Text != first.Text {
		t.Fatalf("second pass changed text: %q -> %q", first.Text, second.Text)
	}
}

func TestRedact_Empty(t *testing.T) {
	r := Redact("")
	if r.Text != "" || r.Total() != 0 {
		t.Fatalf("empty input: got text=%q total=%d", r.Text, r.Total())
	}
}

func TestResult_Names_Sorted(t *testing.T) {
	in := "AKIAIOSFODNN7EXAMPLE and ghp_" + strings.Repeat("x", 36)
	r := Redact(in)
	names := r.Names()
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Fatalf("names not sorted: %v", names)
		}
	}
}

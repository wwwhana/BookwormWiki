package main

import "testing"

func TestDisplayEnvValueNeverRevealsCredentials(t *testing.T) {
	for _, test := range []struct {
		name  string
		value string
		want  string
	}{
		{name: "BOOKWORMWIKI_OPENAI_API_KEY", value: "x", want: "<set>"},
		{name: "BOOKWORMWIKI_AUTH_API_SECRET", value: "secret-value", want: "<set>"},
		{name: "BOOKWORMWIKI_ADMIN_TOKEN", value: "token-value", want: "<set>"},
		{name: "BOOKWORMWIKI_POSTGRES_DSN", value: "postgresql://user:password@example/db", want: "<set>"},
		{name: "BOOKWORMWIKI_AUTH_API_SECRET", value: "", want: "<empty>"},
		{name: "BOOKWORMWIKI_LOG_LEVEL", value: "debug", want: "debug"},
	} {
		t.Run(test.name+test.want, func(t *testing.T) {
			if got := displayEnvValue(test.name, test.value); got != test.want {
				t.Fatalf("displayEnvValue(%q, value) = %q, want %q", test.name, got, test.want)
			}
		})
	}
}

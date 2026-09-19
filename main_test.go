package main

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPrintPasswordHash(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := printPasswordHash(strings.NewReader("correct horse battery staple\r\n"), &out, &errOut); code != 0 {
		t.Fatalf("exit code %d: %s", code, errOut.String())
	}
	hash := strings.TrimSpace(out.String())
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("correct horse battery staple")); err != nil {
		t.Errorf("printed hash does not verify: %v (%q)", err, hash)
	}
	if strings.Contains(errOut.String(), "correct horse") || strings.Contains(out.String(), "correct horse") {
		t.Errorf("the password must never be echoed")
	}

	for name, input := range map[string]string{"too short": "1234567\n", "empty": "", "too long for bcrypt": strings.Repeat("x", 73) + "\n"} {
		out.Reset()
		if code := printPasswordHash(strings.NewReader(input), &out, &errOut); code == 0 || out.Len() != 0 {
			t.Errorf("%s: want a failure without output, got code %d and %q", name, code, out.String())
		}
	}
}

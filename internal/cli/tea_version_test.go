package cli

import (
	"errors"
	"strings"
	"testing"
)

func TestTeaLoginErrorDetectsV1(t *testing.T) {
	orig := runOut
	defer func() { runOut = orig }()

	cases := []struct {
		version string
		want    string
		notWant string
	}{
		{"tea version v1.3.0", "tea v1.x is not supported yet", "tea login add"},
		{"tea version 1.3.0", "tea v1.x is not supported yet", "tea login add"},
		{"tea version v0.16.0", "no tea login for git.example.com", ""},
		{"tea version 0.9.4", "no tea login for git.example.com", ""},
		{"", "no tea login for git.example.com", ""},
		{"unknown", "no tea login for git.example.com", ""},
	}
	for _, tc := range cases {
		runOut = func(name string, args ...string) (string, error) {
			return tc.version, nil
		}
		err := teaLoginError("git.example.com")
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("version %q: got %v, want %q 포함", tc.version, err, tc.want)
		}
		if tc.notWant != "" && strings.Contains(err.Error(), tc.notWant) {
			t.Errorf("version %q: %q 문구가 없어야 한다, got %v", tc.version, tc.notWant, err)
		}
	}

	runOut = func(name string, args ...string) (string, error) {
		return "", errors.New("tea not found")
	}
	if err := teaLoginError("git.example.com"); !strings.Contains(err.Error(), "no tea login") {
		t.Errorf("버전 조회 실패 시 기존 안내 기대, got %v", err)
	}
}

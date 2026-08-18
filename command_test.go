package main

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseRequest(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want Request
	}{
		{name: "repo 생략 list", args: []string{"list", "--limit", "5"},
			want: Request{Resource: "repo", Action: "list", Limit: "5"}},
		{name: "repo 명시 list", args: []string{"repo", "list"},
			want: Request{Resource: "repo", Action: "list"}},
		{name: "전역 repo flag", args: []string{"--repo", "https://github.com/o/r", "view"},
			want: Request{Resource: "repo", Action: "view", RepoFlag: "https://github.com/o/r"}},
		{name: "후행 repo flag", args: []string{"issue", "list", "--repo", "https://github.com/o/r"},
			want: Request{Resource: "issue", Action: "list", RepoFlag: "https://github.com/o/r"}},
		{name: "issue view", args: []string{"issue", "view", "42"},
			want: Request{Resource: "issue", Action: "view", Number: "42"}},
		{name: "issue create", args: []string{"issue", "create", "--title", "t", "--body", "b"},
			want: Request{Resource: "issue", Action: "create", Title: "t", Body: "b"}},
		{name: "pr list state", args: []string{"pr", "list", "--state", "all", "--limit", "3"},
			want: Request{Resource: "pr", Action: "list", State: "all", Limit: "3"}},
		{name: "pr create full", args: []string{"pr", "create", "--title", "t", "--body", "b", "--base", "main", "--head", "f", "--draft"},
			want: Request{Resource: "pr", Action: "create", Title: "t", Body: "b", Base: "main", Head: "f", Draft: true}},
		{name: "repo create", args: []string{"--repo", "https://gitea.com/o/r", "create", "--private", "--description", "d"},
			want: Request{Resource: "repo", Action: "create", RepoFlag: "https://gitea.com/o/r", Private: true, Description: "d"}},
		{name: "clone dir", args: []string{"clone", "https://github.com/o/r", "dst"},
			want: Request{Resource: "repo", Action: "clone", CloneURL: "https://github.com/o/r", CloneDir: "dst"}},
		{name: "pull 전달", args: []string{"pull", "--rebase", "origin", "main"},
			want: Request{Resource: "repo", Action: "pull", GitArgs: []string{"--rebase", "origin", "main"}}},
		{name: "push 전달", args: []string{"repo", "push", "--force-with-lease"},
			want: Request{Resource: "repo", Action: "push", GitArgs: []string{"--force-with-lease"}}},
	}
	for _, c := range cases {
		got, err := ParseRequest(c.args)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s = %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestParseRequestErrors(t *testing.T) {
	bad := [][]string{
		{},                                   // 명령 없음
		{"unknown"},                          // 알 수 없는 자원
		{"issue"},                            // action 없음
		{"issue", "close", "1"},              // 지원 안 하는 action
		{"issue", "view"},                    // number 없음
		{"issue", "view", "1", "2"},          // 인자 초과
		{"issue", "list", "--wat"},           // 알 수 없는 flag
		{"pr", "list", "--state", "merged"},  // 지원 안 하는 state
		{"pr", "create", "--title"},          // 값 없는 flag
		{"clone"},                            // URL 없음
		{"clone", "u", "d", "x"},             // 인자 초과
		{"create", "--public"},               // --repo 없는 repo create
		{"create", "--repo", "https://x.com/o/r"},                 // 공개 범위 없음
		{"create", "--repo", "https://x.com/o/r", "--public", "--private"}, // 둘 다 지정
		{"list", "extra"},                    // list에 positional
	}
	for _, args := range bad {
		_, err := ParseRequest(args)
		var ue UsageError
		if !errors.As(err, &ue) {
			t.Errorf("ParseRequest(%v): UsageError 기대, got %v", args, err)
		}
	}
}

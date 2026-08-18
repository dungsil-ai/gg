package main

import (
	"fmt"
	"net/url"
	"strings"
)

type Provider string

const (
	GH   Provider = "gh"
	GLab Provider = "glab"
	Tea  Provider = "tea"
)

// RepoURL은 저장소 URL에서 얻은 forge 좌표다. Host는 lowercase, port 없음.
type RepoURL struct {
	Host  string
	Owner string // GitLab namespace는 "grp/sub"처럼 /를 포함할 수 있다
	Name  string // 끝의 .git 제거
}

func (r RepoURL) Slug() string  { return r.Owner + "/" + r.Name }
func (r RepoURL) HTTPS() string { return "https://" + r.Host + "/" + r.Slug() }

// ParseRepoURL은 https/ssh/SCP 형식 저장소 URL을 파싱한다.
func ParseRepoURL(raw string) (RepoURL, error) {
	raw = strings.TrimSpace(raw)
	var host, path string
	switch {
	case strings.HasPrefix(raw, "https://"), strings.HasPrefix(raw, "http://"), strings.HasPrefix(raw, "ssh://"):
		u, err := url.Parse(raw)
		if err != nil {
			return RepoURL{}, fmt.Errorf("invalid repository URL %q: %v", raw, err)
		}
		host = u.Hostname()
		path = u.Path
	case raw == "", strings.Contains(raw, "://"):
		return RepoURL{}, fmt.Errorf("unsupported repository URL %q", raw)
	default: // SCP 형식: [user@]host:owner/repo
		i := strings.Index(raw, ":")
		if i < 0 {
			return RepoURL{}, fmt.Errorf("unsupported repository URL %q", raw)
		}
		host = raw[:i]
		if at := strings.Index(host, "@"); at >= 0 {
			host = host[at+1:]
		}
		path = raw[i+1:]
	}
	host = strings.ToLower(host)
	var segs []string
	for _, s := range strings.Split(path, "/") {
		if s != "" {
			segs = append(segs, s)
		}
	}
	if host == "" || len(segs) < 2 {
		return RepoURL{}, fmt.Errorf("repository URL %q must look like host/owner/repo", raw)
	}
	name := strings.TrimSuffix(segs[len(segs)-1], ".git")
	if name == "" {
		return RepoURL{}, fmt.Errorf("repository URL %q has an empty repository name", raw)
	}
	return RepoURL{Host: host, Owner: strings.Join(segs[:len(segs)-1], "/"), Name: name}, nil
}

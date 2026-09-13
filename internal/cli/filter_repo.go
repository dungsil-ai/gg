package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// 이 파일은 외부 git-filter-repo 바이너리 없이 gg가 직접 수행하는 히스토리
// 재작성 로직이다. git fast-export로 전체 히스토리를 받아 Go에서 경로·내용·
// 저자 필터를 적용한 뒤 git fast-import로 다시 쓴다.

// setFilterRepo는 파싱 단계의 사용법 오류만 검증한다. 파일 존재나 정규식
// 컴파일 같은 의미 검증은 실행 단계(runFilterRepo)에서 usageErr로 보고한다.
func setFilterRepo(req *Request, pos []string) error {
	if req.RepoFlag != "" {
		return usageErr("--repo is not supported for repo filter-repo")
	}
	if len(req.FilterPaths) == 0 && len(req.FilterRenames) == 0 &&
		len(req.FilterReplaces) == 0 && req.FilterMailmap == "" {
		return usageErr("repo filter-repo needs at least one of --path, --path-rename, --replace-text, or --mailmap")
	}
	if req.FilterInvert && len(req.FilterPaths) == 0 {
		return usageErr("repo filter-repo --invert-paths needs --path")
	}
	for _, p := range req.FilterPaths {
		if strings.TrimSpace(p) == "" {
			return usageErr("repo filter-repo --path needs a non-empty value")
		}
	}
	for _, r := range req.FilterRenames {
		old, new, ok := strings.Cut(r, ":")
		if !ok || strings.TrimSpace(old) == "" || strings.TrimSpace(new) == "" {
			return usageErr("repo filter-repo --path-rename must look like <old:new>")
		}
	}
	for _, r := range req.FilterReplaces {
		if strings.TrimSpace(r) == "" {
			return usageErr("repo filter-repo --replace-text needs a non-empty value")
		}
	}
	if req.FilterMailmap != "" && strings.TrimSpace(req.FilterMailmap) == "" {
		return usageErr("repo filter-repo --mailmap needs a file")
	}
	return nil
}

// pathRename은 정규화된 경로 prefix 치환 하나다.
type pathRename struct {
	old, new string
}

// replaceRule은 블롭 내용에 적용할 정규식 치환 하나다.
type replaceRule struct {
	re          *regexp.Regexp
	replacement string
	raw         string
}

// mailmapEntry는 하나의 commit email(선택으로 이름까지)에 대한 교정 값이다.
type mailmapEntry struct {
	name, email string
}

// mailmap은 commit email 기준과 (이름, email) 기준의 두 단계 표다.
// 이름까지 적힌 줄이 email만 적힌 줄보다 먼저 적용된다.
type mailmap struct {
	byEmail     map[string]mailmapEntry
	byNameEmail map[string]mailmapEntry
}

// resolvedFilters는 실행 단계에서 정규화·컴파일을 마친 필터다.
type resolvedFilters struct {
	paths    []string
	invert   bool
	renames  []pathRename
	replaces []replaceRule
	mail     *mailmap
}

// filterStats는 --dry-run 미리보기와 완료 요약에 쓰는 계수다.
type filterStats struct {
	commits        int
	blobs          int
	blobsChanged   int
	identities     int
	identitiesDone int
	pathsDropped   int
	pathsRenamed   int
}

func runFilterRepo(req Request) error {
	if len(req.FilterPaths) == 0 && len(req.FilterRenames) == 0 &&
		len(req.FilterReplaces) == 0 && req.FilterMailmap == "" {
		return usageErr("repo filter-repo needs at least one of --path, --path-rename, --replace-text, or --mailmap")
	}
	if req.FilterInvert && len(req.FilterPaths) == 0 {
		return usageErr("repo filter-repo --invert-paths needs --path")
	}
	if _, err := lookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not on PATH")
	}
	if _, err := runOut("git", "rev-parse", "--git-dir"); err != nil {
		return fmt.Errorf("not a git repository")
	}
	bare := false
	if out, err := runOut("git", "rev-parse", "--is-bare-repository"); err == nil {
		bare = strings.TrimSpace(out) == "true"
	}
	// bare 저장소에는 작업 트리가 없어 git status가 항상 실패한다. 작업 트리
	// 검사는 worktree가 있을 때만 한다 — 그렇지 않으면 bare 저장소는 항상
	// "cannot check working tree"로 거부돼 bare 감지와 복구 생략이 죽은
	// 코드가 된다.
	if !bare {
		if out, err := runOut("git", "status", "--porcelain"); err != nil {
			return fmt.Errorf("cannot check working tree: %w", err)
		} else if strings.TrimSpace(out) != "" {
			return fmt.Errorf("working tree is dirty (commit, stash, or discard changes first)")
		}
		// detached HEAD는 재작성 후에도 예전 커밋을 가리킨다. fast-import는
		// HEAD(raw SHA)를 갱신하지 않고, 뒤따르는 reset --hard가 재작성 전
		// 커밋의 내용을 작업 트리에 되살린다 — 제거하려던 파일이 디스크에
		// 다시 나타나므로 미리 거부한다.
		if _, err := runOut("git", "symbolic-ref", "-q", "HEAD"); err != nil {
			return fmt.Errorf("detached HEAD (check out a branch first); rewriting would leave the pre-rewrite commit checked out")
		}
		// stash 항목은 재작성 전 커밋·블롭을 참조한다. filter-repo는 branch와
		// tag만 다시 쓰므로 stash를 남겨 두면 예전 객체가 gc 뒤에도 유지되고,
		// 재작성 결과와 뒤섞여 의미를 알 수 없게 된다. 먼저 비우도록 요구한다.
		if out, err := runOut("git", "stash", "list"); err == nil && strings.TrimSpace(out) != "" {
			return fmt.Errorf("stash is not empty (drop or pop it first); filter-repo rewrites branches and tags only")
		}
	}
	if !req.FilterDryRun && !req.FilterForce {
		return fmt.Errorf("refusing to rewrite history without --force (a mirror backup is created before rewriting)")
	}
	filters, err := resolveFilterRepoFilters(req)
	if err != nil {
		return err
	}
	stats := &filterStats{}
	if req.FilterDryRun {
		if err := dryRunFilterRepo(filters, stats); err != nil {
			return err
		}
		fmt.Fprintf(osStdout, "dry-run: %d commits, %d blobs (%d would change), %d identities (%d would change), %d paths dropped, %d renamed\n",
			stats.commits, stats.blobs, stats.blobsChanged, stats.identities, stats.identitiesDone, stats.pathsDropped, stats.pathsRenamed)
		return nil
	}
	if remotes, err := runOut("git", "remote"); err == nil && strings.TrimSpace(remotes) != "" {
		fmt.Fprintln(osStderr, "gg: warning: rewriting published history; force-push every rewritten ref after this")
	}
	// README가 보장하는 fresh clone 백업이다. reflog expire과 gc --prune=now가
	// 뒤따르므로 이 백업이 재작성 실수의 유일한 복구 수단이다.
	backupPath, err := backupFilterRepo()
	if err != nil {
		return err
	}
	if err := rewriteFilterRepo(filters, stats); err != nil {
		return err
	}
	// reflog expire은 재작성 대상인 branches·tags로 한정한다. --all은 stash 등
	// 재작성하지 않은 ref의 reflog까지 지워 접근 불가능하게 만든다. reflog가
	// 없는 ref(대표적으로 tag)에 expire를 실행하면 오류가 나므로 존재할 때만
	// 만료한다.
	rewrittenRefs, err := runOut("git", "for-each-ref", "--format=%(refname)", "refs/heads", "refs/tags")
	if err != nil {
		return fmt.Errorf("history rewritten but reflog expire failed: %w", err)
	}
	for _, ref := range strings.Fields(rewrittenRefs) {
		if _, err := runOut("git", "reflog", "exists", ref); err != nil {
			continue
		}
		if _, err := runOut("git", "reflog", "expire", "--expire=now", ref); err != nil {
			return fmt.Errorf("history rewritten but reflog expire failed: %w", err)
		}
	}
	if _, err := runOut("git", "gc", "--prune=now", "--quiet"); err != nil {
		return fmt.Errorf("history rewritten but gc failed: %w", err)
	}
	if !bare {
		if _, err := runOut("git", "reset", "--hard", "-q"); err != nil {
			return fmt.Errorf("history rewritten but working tree reset failed: %w", err)
		}
	}
	fmt.Fprintf(osStdout, "rewrote %d commits, %d blobs changed (backup: %s); force-push rewritten refs (e.g. git push --force --all && git push --force --tags)\n",
		stats.commits, stats.blobsChanged, backupPath)
	return nil
}

// backupFilterRepo는 재작성 전 현재 저장소의 mirror 백업을 만든다. 백업은
// git dir 안의 filter-repo-backup에 두고, 이미 있으면 덮쓰지 않는다 — 실수로
// 이전 백업을 지우지 않도록 사용자가 직접 확인한 뒤 치우게 한다.
func backupFilterRepo() (string, error) {
	gitDir, err := runOut("git", "rev-parse", "--absolute-git-dir")
	if err != nil || gitDir == "" {
		return "", fmt.Errorf("cannot locate git dir for backup: %w", err)
	}
	backup := filepath.Join(gitDir, "filter-repo-backup")
	if _, err := os.Stat(backup); err == nil {
		return "", fmt.Errorf("backup already exists at %s (remove or move it, then re-run)", backup)
	}
	if _, err := runOut("git", "clone", "--mirror", "--no-local", ".", backup); err != nil {
		return "", fmt.Errorf("cannot create backup clone at %s: %w", backup, err)
	}
	return backup, nil
}

func resolveFilterRepoFilters(req Request) (*resolvedFilters, error) {
	f := &resolvedFilters{invert: req.FilterInvert}
	for _, p := range req.FilterPaths {
		n, err := normalizeFilterRepoPath(p)
		if err != nil {
			return nil, usageErr(fmt.Sprintf("repo filter-repo --path %q: %s", p, err.Error()))
		}
		f.paths = append(f.paths, n)
	}
	for _, r := range req.FilterRenames {
		old, new, _ := strings.Cut(r, ":")
		no, err := normalizeFilterRepoPath(old)
		if err != nil {
			return nil, usageErr(fmt.Sprintf("repo filter-repo --path-rename %q: bad old path", r))
		}
		nn, err := normalizeFilterRepoPath(new)
		if err != nil {
			return nil, usageErr(fmt.Sprintf("repo filter-repo --path-rename %q: bad new path", r))
		}
		if no == nn {
			return nil, usageErr(fmt.Sprintf("repo filter-repo --path-rename %q: old and new are the same", r))
		}
		f.renames = append(f.renames, pathRename{old: no, new: nn})
	}
	replaces, err := resolveReplaceTextFilters(req.FilterReplaces)
	if err != nil {
		return nil, err
	}
	f.replaces = replaces
	if req.FilterMailmap != "" {
		mm, err := parseMailmapFile(req.FilterMailmap)
		if err != nil {
			return nil, err
		}
		f.mail = mm
	}
	return f, nil
}

// normalizeFilterRepoPath는 저장소 상대 경로를 정규화한다. 디렉터리 지정은
// prefix로 취급하므로 끝의 /만 걷는다.
func normalizeFilterRepoPath(p string) (string, error) {
	p = strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimPrefix(p, "/")
	p = strings.TrimSuffix(p, "/")
	if p == "" || p == "." {
		return "", fmt.Errorf("empty path")
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "", fmt.Errorf("invalid path")
		}
	}
	return path.Clean(p), nil
}

// matchFilterRepoPath는 파일이 --path 패턴 중 하나에 걸리는지 본다.
// 패턴이 디렉터리면 그 아래 전체가 걸린다.
func matchFilterRepoPath(file string, patterns []string) bool {
	for _, p := range patterns {
		if file == p || strings.HasPrefix(file, p+"/") {
			return true
		}
	}
	return false
}

// keepFilterRepoPath는 경로 필터 뒤에 파일을 남길지 정한다.
func keepFilterRepoPath(f *resolvedFilters, file string) bool {
	if len(f.paths) == 0 {
		return true
	}
	hit := matchFilterRepoPath(file, f.paths)
	if f.invert {
		return !hit
	}
	return hit
}

// renameFilterRepoPath는 남긴 파일에 prefix 치환을 적용한다. 처음 걸린
// 규칙 하나만 적용하고 두 번째부터는 건너뛴다.
func renameFilterRepoPath(f *resolvedFilters, file string) (string, bool) {
	for _, r := range f.renames {
		if file == r.old {
			return r.new, true
		}
		if strings.HasPrefix(file, r.old+"/") {
			return r.new + file[len(r.old):], true
		}
	}
	return file, false
}

// resolveReplaceTextFilters는 --replace-text 값을 인라인(regex==>replacement)
// 또는 파일 경로로 풀어 컴파일한다. 파일은 한 줄에 하나씩 regex==>replacement
// 형식을 쓰고 빈 줄과 # 주석을 건너뛴다.
func resolveReplaceTextFilters(raw []string) ([]replaceRule, error) {
	var out []replaceRule
	for _, v := range raw {
		if st, err := os.Stat(v); err == nil && !st.IsDir() {
			data, err := os.ReadFile(v)
			if err != nil {
				return nil, usageErr(fmt.Sprintf("repo filter-repo --replace-text: cannot read file %q", v))
			}
			for i, line := range strings.Split(string(data), "\n") {
				line = strings.TrimRight(line, "\r")
				if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
					continue
				}
				rule, err := compileReplaceRule(line)
				if err != nil {
					return nil, usageErr(fmt.Sprintf("repo filter-repo --replace-text file %q line %d: %s", v, i+1, err.Error()))
				}
				out = append(out, rule)
			}
			continue
		}
		rule, err := compileReplaceRule(v)
		if err != nil {
			return nil, usageErr(fmt.Sprintf("repo filter-repo --replace-text %q: %s (want <regex==>replacement> or a file)", v, err.Error()))
		}
		out = append(out, rule)
	}
	return out, nil
}

func compileReplaceRule(s string) (replaceRule, error) {
	expr, repl, ok := strings.Cut(s, "==>")
	if !ok {
		return replaceRule{}, fmt.Errorf("missing ==> separator")
	}
	if strings.TrimSpace(expr) == "" {
		return replaceRule{}, fmt.Errorf("empty regex")
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return replaceRule{}, fmt.Errorf("bad regex: %w", err)
	}
	return replaceRule{re: re, replacement: repl, raw: s}, nil
}

// apply는 블롭 내용에 규칙을 적용한다. 치환값은 문자 그대로 넣는다 —
// ReplaceAllString은 치환값의 $1·${name}을 캡처 확장으로 읽으므로, $가 든
// 치환값(가격, 셸 변수 등)이 조용히 깨진다. git-filter-repo도 문자 그대로
// 넣는다.
func (r replaceRule) apply(s string) string {
	return r.re.ReplaceAllStringFunc(s, func(string) string { return r.replacement })
}

var mailmapEmailRe = regexp.MustCompile(`<([^<>]*)>`)

// parseMailmapFile은 자주 쓰는 mailmap 줄 형식을 읽는다:
// "Name <proper> <commit>", "Name <proper> Commit <commit>",
// "<proper> <commit>", "Name <proper>".
func parseMailmapFile(p string) (*mailmap, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, usageErr(fmt.Sprintf("repo filter-repo --mailmap: cannot read file %q", p))
	}
	mm := &mailmap{byEmail: map[string]mailmapEntry{}, byNameEmail: map[string]mailmapEntry{}}
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		locs := mailmapEmailRe.FindAllStringSubmatchIndex(line, -1)
		if len(locs) == 0 || len(locs) > 2 {
			return nil, usageErr(fmt.Sprintf("repo filter-repo --mailmap file %q line %d: bad mailmap line", p, i+1))
		}
		emails := make([]string, len(locs))
		for k, m := range locs {
			emails[k] = strings.TrimSpace(line[m[2]:m[3]])
		}
		properName := strings.TrimSpace(line[:locs[0][0]])
		if len(locs) == 1 {
			if emails[0] == "" {
				return nil, usageErr(fmt.Sprintf("repo filter-repo --mailmap file %q line %d: empty email", p, i+1))
			}
			mm.byEmail[strings.ToLower(emails[0])] = mailmapEntry{name: properName, email: emails[0]}
			continue
		}
		commitName := strings.TrimSpace(line[locs[0][1]:locs[1][0]])
		if emails[1] == "" {
			return nil, usageErr(fmt.Sprintf("repo filter-repo --mailmap file %q line %d: empty commit email", p, i+1))
		}
		e := mailmapEntry{name: properName, email: emails[0]}
		if commitName == "" {
			mm.byEmail[strings.ToLower(emails[1])] = e
		} else {
			mm.byNameEmail[strings.ToLower(commitName)+"\x00"+strings.ToLower(emails[1])] = e
		}
	}
	return mm, nil
}

// applyMailmap은 이름·email에 mailmap을 적용한다. 바뀌면 true를 돌려준다.
func applyMailmap(mm *mailmap, name, email string) (string, string, bool) {
	if mm == nil {
		return name, email, false
	}
	if e, ok := mm.byNameEmail[strings.ToLower(name)+"\x00"+strings.ToLower(email)]; ok {
		nn, ee := name, email
		if e.name != "" {
			nn = e.name
		}
		if e.email != "" {
			ee = e.email
		}
		return nn, ee, nn != name || ee != email
	}
	if e, ok := mm.byEmail[strings.ToLower(email)]; ok {
		nn, ee := name, email
		if e.name != "" {
			nn = e.name
		}
		if e.email != "" {
			ee = e.email
		}
		return nn, ee, nn != name || ee != email
	}
	return name, email, false
}

// epoch는 음수일 수 있다(1970년 이전 타임스탬프, 예: GIT_COMMITTER_DATE="@-86400").
// 양수만 받으면 그런 identity 줄이 mailmap 대상에서 조용히 빠진다.
var identityRe = regexp.MustCompile(`^(author|committer|tagger) (.*) <([^>]*)> (-?\d+) ([+-]\d{4})$`)

// rewriteIdentityLine은 fast-export의 author/committer/tagger 줄을 mailmap
// 기준으로 고친다.
func rewriteIdentityLine(f *resolvedFilters, line string, stats *filterStats) string {
	m := identityRe.FindStringSubmatch(line)
	if m == nil {
		return line
	}
	if stats != nil {
		stats.identities++
	}
	nn, ee, changed := applyMailmap(f.mail, m[2], m[3])
	if !changed {
		return line
	}
	if stats != nil {
		stats.identitiesDone++
	}
	return m[1] + " " + nn + " <" + ee + "> " + m[4] + " " + m[5]
}

// parseFastExportPath는 fast-export/fast-import의 경로 토큰을 푼다.
// 공백·따옴표가 든 경로는 C 인용으로 오므로 Unquote를 시도한다.
func parseFastExportPath(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' {
		if u, err := strconv.Unquote(s); err == nil {
			return u
		}
	}
	return s
}

// quoteFastExportPath는 공백·따옴표·제어문자가 든 경로만 인용한다. git의
// unquote_c_style은 \xNN·\uNNNN 같은 Go 확장 이스케이프를 모르므로(모르는
// 시퀀스는 문자 그대로 남는다) strconv.Quote 대신 git이 인식하는 C 인용
// (\a \b \f \n \r \t \v \" \\와 8진수)으로 쓴다.
func quoteFastExportPath(s string) string {
	need := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c <= ' ' || c == '"' || c == '\\' || c == 0x7f {
			need = true
			break
		}
	}
	if !need {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\a':
			b.WriteString(`\a`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\v':
			b.WriteString(`\v`)
		default:
			if c < ' ' || c == 0x7f {
				fmt.Fprintf(&b, "\\%03o", c)
			} else {
				// 0x80 이상 바이트(UTF-8 포함, 비UTF-8 포함)는 그대로 둔다.
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// filterRepoExportArgs는 재작성 대상 ref 범위다. --all은 refs/stash와
// refs/remotes까지 재작성해 stash 접근을 깨고 원격 추적 ref를 재작성된 SHA로
// 어긋나게 만든다. filter-repo의 대상은 branch와 tag뿐이다.
var filterRepoExportArgs = []string{"fast-export", "--branches", "--tags", "--full-tree"}

// dryRunFilterRepo는 fast-export를 읽어 필터 통계만 낸다.
func dryRunFilterRepo(f *resolvedFilters, stats *filterStats) error {
	gitPath, err := lookPath("git")
	if err != nil {
		return fmt.Errorf("git is not installed or not on PATH")
	}
	cmd := exec.Command(gitPath, filterRepoExportArgs...)
	cmd.Stderr = os.Stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("git fast-export failed: %w", err)
	}
	return runDryFilterPipeline(out, func() { _ = cmd.Process.Kill() }, func() error { return cmd.Wait() }, f, stats)
}

// runDryFilterPipeline은 dry-run 스트림을 필터링한다. 필터가 중간에 실패하면
// fast-export를 죽인다 — 읽기를 멈춘 채 Wait만 하면 fast-export가 가득 찬
// 파이프에 막혀 gg가 영원히 끝나지 않는다.
func runDryFilterPipeline(exportOut io.Reader, kill func(), wait func() error, f *resolvedFilters, stats *filterStats) error {
	ferr := filterFastExportStream(exportOut, io.Discard, f, stats, true)
	if ferr != nil && kill != nil {
		kill()
	}
	werr := wait()
	if ferr != nil {
		return ferr
	}
	if werr != nil {
		return fmt.Errorf("git fast-export failed: %w", werr)
	}
	return nil
}

// rewriteFilterRepo는 fast-export 출력을 필터링해 fast-import에 바로 넣는다.
func rewriteFilterRepo(f *resolvedFilters, stats *filterStats) error {
	gitPath, err := lookPath("git")
	if err != nil {
		return fmt.Errorf("git is not installed or not on PATH")
	}
	exportCmd := exec.Command(gitPath, filterRepoExportArgs...)
	exportCmd.Stderr = os.Stderr
	exportOut, err := exportCmd.StdoutPipe()
	if err != nil {
		return err
	}
	importCmd := exec.Command(gitPath, "fast-import", "--force", "--quiet")
	importCmd.Stderr = os.Stderr
	importIn, err := importCmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := exportCmd.Start(); err != nil {
		return fmt.Errorf("git fast-export failed: %w", err)
	}
	if err := importCmd.Start(); err != nil {
		_ = exportCmd.Wait()
		return fmt.Errorf("git fast-import failed: %w", err)
	}
	// fast-import는 stdin이 깨끗이 닫히면(EOF) 지금까지 읽은 ref를 확정한다.
	// 필터가 중간에 실패했으면 부분 재작성을 남기지 않도록 닫지 말고 죽인다.
	killAll := func() {
		if importCmd.Process != nil {
			_ = importCmd.Process.Kill()
		}
		if exportCmd.Process != nil {
			_ = exportCmd.Process.Kill()
		}
	}
	return runFilterPipeline(exportOut, importIn, killAll,
		func() error { return exportCmd.Wait() },
		func() error { return importCmd.Wait() }, f, stats)
}

// runFilterPipeline은 export 스트림을 필터링해 import로 옮긴다. 필터 오류 시
// killAll로 두 프로세스를 끝내므로 fast-import의 부분 확정과 fast-export의
// 파이프 교착을 막는다. 오류가 있으면 filter 오류가 최우선으로 반환된다.
func runFilterPipeline(exportOut io.Reader, importIn io.WriteCloser, killAll func(), waitExport, waitImport func() error, f *resolvedFilters, stats *filterStats) error {
	filterErr := filterFastExportStream(exportOut, importIn, f, stats, false)
	if filterErr != nil && killAll != nil {
		killAll()
	}
	_ = importIn.Close()
	exportWaitErr := waitExport()
	importWaitErr := waitImport()
	if filterErr != nil {
		return filterErr
	}
	if exportWaitErr != nil {
		return fmt.Errorf("git fast-export failed: %w", exportWaitErr)
	}
	if importWaitErr != nil {
		return fmt.Errorf("git fast-import failed: %w", importWaitErr)
	}
	return nil
}

// filterFastExportStream은 fast-export 스트림을 읽어 필터를 적용한 뒤
// fast-import 스트림으로 쓴다. data 섹션은 바이트 길이대로 정확히 옮긴다.
func filterFastExportStream(r io.Reader, w io.Writer, f *resolvedFilters, stats *filterStats, dryRun bool) error {
	rd := &fastExportReader{br: bufio.NewReader(r)}
	bw := bufio.NewWriter(w)
	defer func() { _ = bw.Flush() }()
	for {
		line, err := rd.next()
		if err == io.EOF {
			return bw.Flush()
		}
		if err != nil {
			return err
		}
		switch {
		case line == "blob":
			if err := filterBlob(rd.br, bw, f, stats, dryRun); err != nil {
				return err
			}
		case strings.HasPrefix(line, "commit "):
			if err := filterCommit(rd, bw, line, f, stats, dryRun); err != nil {
				return err
			}
		case strings.HasPrefix(line, "tag "):
			if err := filterTag(rd, bw, line, f, stats, dryRun); err != nil {
				return err
			}
		case strings.HasPrefix(line, "reset "):
			if !dryRun {
				if _, err := bw.WriteString(line + "\n"); err != nil {
					return err
				}
			}
		case line == "done":
			if !dryRun {
				if _, err := bw.WriteString(line + "\n"); err != nil {
					return err
				}
			}
			return bw.Flush()
		default:
			// progress 같은 보조 줄은 그대로 둔다.
			if !dryRun {
				if _, err := bw.WriteString(line + "\n"); err != nil {
					return err
				}
			}
		}
	}
}

// fastExportReader는 최상위 키워드 한 줄 미리 읽기를 되돌리는 판독기다.
// filterCommit/filterTag가 다음 블록의 첫 줄을 앞서 읽으므로, 상위 루프가
// 그 줄을 다시 보게 되밀기 슬롯으로 넘긴다.
type fastExportReader struct {
	br     *bufio.Reader
	pushed []string
}

func (rd *fastExportReader) next() (string, error) {
	if len(rd.pushed) > 0 {
		line := rd.pushed[len(rd.pushed)-1]
		rd.pushed = rd.pushed[:len(rd.pushed)-1]
		return line, nil
	}
	return readFastExportLine(rd.br)
}

func (rd *fastExportReader) pushBack(line string) {
	rd.pushed = append(rd.pushed, line)
}

func readFastExportLine(br *bufio.Reader) (string, error) {
	s, err := br.ReadString('\n')
	if err != nil {
		if err == io.EOF && s != "" {
			return strings.TrimSuffix(s, "\n"), nil
		}
		return "", err
	}
	return strings.TrimSuffix(strings.TrimSuffix(s, "\n"), "\r"), nil
}

// nextFastExportLine은 되밀린 최상위 줄을 먼저 돌려준다.
func nextFastExportLine(rd *fastExportReader) (string, error) {
	return rd.next()
}

func writeFastExportLine(bw *bufio.Writer, s string) error {
	_, err := bw.WriteString(s + "\n")
	return err
}

// readDataSection은 "data <len>" 줄 다음의 날 바이트를 정확히 읽는다.
func readDataSection(br *bufio.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(br, buf)
	return buf, err
}

// parseDataLength는 "data 123" 줄에서 길이를 읽는다.
func parseDataLength(line string) (int, bool) {
	if !strings.HasPrefix(line, "data ") {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "data ")))
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func filterBlob(br *bufio.Reader, bw *bufio.Writer, f *resolvedFilters, stats *filterStats, dryRun bool) error {
	if stats != nil {
		stats.blobs++
	}
	markLine, err := readFastExportLine(br)
	if err != nil {
		return err
	}
	dataLine, err := readFastExportLine(br)
	if err != nil {
		return err
	}
	n, ok := parseDataLength(dataLine)
	if !ok {
		return fmt.Errorf("bad fast-export blob: %q", dataLine)
	}
	data, err := readDataSection(br, n)
	if err != nil {
		return err
	}
	changed := false
	if len(f.replaces) > 0 {
		s := string(data)
		for _, r := range f.replaces {
			ns := r.apply(s)
			if ns != s {
				changed = true
				s = ns
			}
		}
		if changed {
			data = []byte(s)
		}
	}
	if changed && stats != nil {
		stats.blobsChanged++
	}
	if dryRun {
		return nil
	}
	if err := writeFastExportLine(bw, "blob"); err != nil {
		return err
	}
	if err := writeFastExportLine(bw, markLine); err != nil {
		return err
	}
	if err := writeFastExportLine(bw, "data "+strconv.Itoa(len(data))); err != nil {
		return err
	}
	_, err = bw.Write(data)
	return err
}

func filterCommit(rd *fastExportReader, bw *bufio.Writer, first string, f *resolvedFilters, stats *filterStats, dryRun bool) error {
	if stats != nil {
		stats.commits++
	}
	var out bytes.Buffer
	tmp := bufio.NewWriter(&out)
	if err := writeFastExportLine(tmp, first); err != nil {
		return err
	}
	br := rd.br
	for {
		line, err := readFastExportLine(br)
		if err == io.EOF {
			// fast-export는 마지막 블록 뒤에 done 없이 끝난다.
			break
		}
		if err != nil {
			return err
		}
		// 다음 최상위 블록이면 되돌릴 수 없으므로 버퍼에 쌓인 커밋을 먼저
		// 내보내고 이 줄을 상위 루프가 이어받도록 되밀기 스택에 넣는다.
		if line == "blob" || strings.HasPrefix(line, "commit ") ||
			strings.HasPrefix(line, "tag ") || strings.HasPrefix(line, "reset ") ||
			line == "done" {
			rd.pushBack(line)
			break
		}
		switch {
		case strings.HasPrefix(line, "author ") || strings.HasPrefix(line, "committer "):
			line = rewriteIdentityLine(f, line, stats)
			if err := writeFastExportLine(tmp, line); err != nil {
				return err
			}
		case strings.HasPrefix(line, "data "):
			n, ok := parseDataLength(line)
			if !ok {
				return fmt.Errorf("bad fast-export commit message: %q", line)
			}
			msg, err := readDataSection(br, n)
			if err != nil {
				return err
			}
			if err := writeFastExportLine(tmp, "data "+strconv.Itoa(len(msg))); err != nil {
				return err
			}
			if _, err := tmp.Write(msg); err != nil {
				return err
			}
		case strings.HasPrefix(line, "M "):
			kept, renamed, rewritten := filterFileLine(f, line, stats)
			if !kept {
				continue
			}
			if renamed {
				line = rewritten
			}
			if err := writeFastExportLine(tmp, line); err != nil {
				return err
			}
		case strings.HasPrefix(line, "D "):
			kept, renamed, rewritten := filterFileLine(f, line, stats)
			if !kept {
				continue
			}
			if renamed {
				line = rewritten
			}
			if err := writeFastExportLine(tmp, line); err != nil {
				return err
			}
		case strings.HasPrefix(line, "R ") || strings.HasPrefix(line, "C "):
			kept, _, rewritten := filterRenameCopyLine(f, line, stats)
			if !kept {
				continue
			}
			if err := writeFastExportLine(tmp, rewritten); err != nil {
				return err
			}
		default:
			// mark, from, merge, deleteall 등은 그대로 둔다.
			if err := writeFastExportLine(tmp, line); err != nil {
				return err
			}
		}
	}
	if err := tmp.Flush(); err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	_, err := bw.Write(out.Bytes())
	return err
}

// filterFileLine은 M/D 줄 하나에 경로 필터와 치환을 적용한다.
func filterFileLine(f *resolvedFilters, line string, stats *filterStats) (kept, renamed bool, rewritten string) {
	var file string
	isDelete := strings.HasPrefix(line, "D ")
	if isDelete {
		file = parseFastExportPath(strings.TrimPrefix(line, "D "))
	} else {
		// M <mode> <dataref> <path>: path에 공백이 있을 수 있어 뒤에서 자른다.
		rest := strings.TrimPrefix(line, "M ")
		parts := strings.SplitN(rest, " ", 3)
		if len(parts) != 3 {
			return true, false, line
		}
		file = parseFastExportPath(parts[2])
	}
	if !keepFilterRepoPath(f, file) {
		if stats != nil {
			stats.pathsDropped++
		}
		return false, false, ""
	}
	newFile, did := renameFilterRepoPath(f, file)
	if did {
		if stats != nil {
			stats.pathsRenamed++
		}
		if isDelete {
			return true, true, "D " + quoteFastExportPath(newFile)
		}
		rest := strings.TrimPrefix(line, "M ")
		parts := strings.SplitN(rest, " ", 3)
		return true, true, "M " + parts[0] + " " + parts[1] + " " + quoteFastExportPath(newFile)
	}
	return true, false, line
}

// filterRenameCopyLine은 R/C 줄의 목적지 기준으로 남길지 정한다.
func filterRenameCopyLine(f *resolvedFilters, line string, stats *filterStats) (bool, bool, string) {
	kind := "R "
	if strings.HasPrefix(line, "C ") {
		kind = "C "
	}
	rest := strings.TrimPrefix(line, kind)
	// 경로는 인용될 수 있어 단순 SplitN으로는 깨진다. 인용을 고려해 뒤에서 푼다.
	old, new := splitQuotedPair(rest)
	if old == "" && new == "" {
		return true, false, line
	}
	if !keepFilterRepoPath(f, new) {
		if stats != nil {
			stats.pathsDropped++
		}
		return false, false, ""
	}
	newOld, didOld := renameFilterRepoPath(f, old)
	newNew, didNew := renameFilterRepoPath(f, new)
	if didOld || didNew {
		if stats != nil {
			stats.pathsRenamed++
		}
		return true, true, kind + quoteFastExportPath(newOld) + " " + quoteFastExportPath(newNew)
	}
	return true, false, line
}

// splitQuotedPair는 "old new" 쌍을 인용을 고려해 나눈다.
func splitQuotedPair(s string) (string, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	if strings.HasPrefix(s, "\"") {
		end := findQuoteEnd(s)
		if end < 0 {
			return "", ""
		}
		first := parseFastExportPath(s[:end+1])
		second := parseFastExportPath(strings.TrimSpace(s[end+1:]))
		return first, second
	}
	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 {
		return "", ""
	}
	// 두 번째가 인용이면 그대로, 첫 번째는 공백이 없어야 한다.
	return strings.TrimSpace(parts[0]), parseFastExportPath(strings.TrimSpace(parts[1]))
}

func findQuoteEnd(s string) int {
	esc := false
	for i := 1; i < len(s); i++ {
		c := s[i]
		if esc {
			esc = false
			continue
		}
		if c == '\\' {
			esc = true
			continue
		}
		if c == '"' {
			return i
		}
	}
	return -1
}

func filterTag(rd *fastExportReader, bw *bufio.Writer, first string, f *resolvedFilters, stats *filterStats, dryRun bool) error {
	var out bytes.Buffer
	tmp := bufio.NewWriter(&out)
	if err := writeFastExportLine(tmp, first); err != nil {
		return err
	}
	br := rd.br
	for {
		line, err := readFastExportLine(br)
		if err == io.EOF {
			// fast-export는 마지막 블록 뒤에 done 없이 끝난다.
			break
		}
		if err != nil {
			return err
		}
		if line == "blob" || strings.HasPrefix(line, "commit ") ||
			strings.HasPrefix(line, "tag ") || strings.HasPrefix(line, "reset ") ||
			line == "done" {
			rd.pushBack(line)
			break
		}
		if strings.HasPrefix(line, "tagger ") {
			line = rewriteIdentityLine(f, line, stats)
		}
		if strings.HasPrefix(line, "data ") {
			n, ok := parseDataLength(line)
			if !ok {
				return fmt.Errorf("bad fast-export tag message: %q", line)
			}
			msg, err := readDataSection(br, n)
			if err != nil {
				return err
			}
			if err := writeFastExportLine(tmp, "data "+strconv.Itoa(len(msg))); err != nil {
				return err
			}
			if _, err := tmp.Write(msg); err != nil {
				return err
			}
			continue
		}
		if err := writeFastExportLine(tmp, line); err != nil {
			return err
		}
	}
	if err := tmp.Flush(); err != nil {
		return err
	}
	if dryRun {
		return nil
	}
	_, err := bw.Write(out.Bytes())
	return err
}

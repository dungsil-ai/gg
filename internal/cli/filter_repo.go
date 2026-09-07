package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
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
	if out, err := runOut("git", "status", "--porcelain"); err != nil {
		return fmt.Errorf("cannot check working tree: %w", err)
	} else if strings.TrimSpace(out) != "" {
		return fmt.Errorf("working tree is dirty (commit, stash, or discard changes first)")
	}
	if !req.FilterDryRun && !req.FilterForce {
		return fmt.Errorf("refusing to rewrite history without --force (use a fresh clone backup first, then re-run with --force)")
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
	if err := rewriteFilterRepo(filters, stats); err != nil {
		return err
	}
	if _, err := runOut("git", "reflog", "expire", "--expire=now", "--all"); err != nil {
		return fmt.Errorf("history rewritten but reflog expire failed: %w", err)
	}
	if _, err := runOut("git", "gc", "--prune=now", "--quiet"); err != nil {
		return fmt.Errorf("history rewritten but gc failed: %w", err)
	}
	if !bare {
		if _, err := runOut("git", "reset", "--hard", "-q"); err != nil {
			return fmt.Errorf("history rewritten but working tree reset failed: %w", err)
		}
	}
	fmt.Fprintf(osStdout, "rewrote %d commits, %d blobs changed; force-push rewritten refs (e.g. git push --force --all && git push --force --tags)\n",
		stats.commits, stats.blobsChanged)
	return nil
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

var identityRe = regexp.MustCompile(`^(author|committer|tagger) (.*) <([^>]*)> (\d+) ([+-]\d{4})$`)

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

// quoteFastExportPath는 공백·따옴표·제어문자가 든 경로만 C 인용한다.
func quoteFastExportPath(s string) string {
	if strings.ContainsAny(s, " \t\n\r\"\\") {
		return strconv.Quote(s)
	}
	return s
}

// dryRunFilterRepo는 fast-export를 읽어 필터 통계만 낸다.
func dryRunFilterRepo(f *resolvedFilters, stats *filterStats) error {
	gitPath, err := lookPath("git")
	if err != nil {
		return fmt.Errorf("git is not installed or not on PATH")
	}
	cmd := exec.Command(gitPath, "fast-export", "--all", "--full-tree")
	cmd.Stderr = os.Stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("git fast-export failed: %w", err)
	}
	ferr := filterFastExportStream(out, io.Discard, f, stats, true)
	werr := cmd.Wait()
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
	exportCmd := exec.Command(gitPath, "fast-export", "--all", "--full-tree")
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
	filterErr := filterFastExportStream(exportOut, importIn, f, stats, false)
	_ = importIn.Close()
	exportWaitErr := exportCmd.Wait()
	importWaitErr := importCmd.Wait()
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
			ns := r.re.ReplaceAllString(s, r.replacement)
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

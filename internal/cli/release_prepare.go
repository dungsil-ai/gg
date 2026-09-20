package cli

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// releasePrepareTagRe는 prepare가 받는 유일한 tag 형식이다. release workflow의
// dispatch·verify 단계와 같은 엄격한 형식으로, prepare가 만든 tag가 workflow
// 게이트를 통과할 수 있게 한다(ADR 0005). release.go의 validReleaseTag는
// ldflags 주입 방어용으로 넓게 받는 별개의 검증이다.
var releasePrepareTagRe = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// runReleasePrepare는 ADR 0001의 릴리즈 의식 — 빈 릴리즈 커밋, HEAD의
// annotated tag, atomic push — 를 검사하고 실행하는 gg-native action이다.
// 모든 검사는 변경 전에 실행하고 하나라도 실패하면 아무 것도 만들지 않는다
// (fail-closed). 자식 실행은 execChild의 stdio·exit code 보존 계약을 재사용한다.
func runReleasePrepare(req Request) error {
	tag := req.Tag
	if req.Explain {
		fmt.Fprintf(osStdout, "release prepare %s\n", tag)
		fmt.Fprintln(osStdout, "검사: tag 형식(vMAJOR.MINOR.PATCH) → working tree → 현재 branch(default branch) → 로컬 tag 부재 → HEAD==원격 tip → 원격 tag 부재")
		fmt.Fprintln(osStdout, "git commit --no-gpg-sign --allow-empty -m \"release: <tag>\"")
		fmt.Fprintln(osStdout, "git tag -a <tag> -m \"Release <tag>\"")
		fmt.Fprintln(osStdout, "git push --atomic origin HEAD:refs/heads/<default> refs/tags/<tag>")
		return nil
	}

	// 원격 조회를 포함한 모든 검사와 실행에 git이 필요하다.
	if _, err := lookPath("git"); err != nil {
		return exitCodeError{Code: 127, Msg: "git is not installed or not on PATH"}
	}

	// 검사 1: tag 형식. workflow 게이트와 같은 엄격한 형식만 받는다(ADR 0005).
	if !releasePrepareTagRe.MatchString(tag) {
		return fmt.Errorf("invalid release tag %q; want vMAJOR.MINOR.PATCH", tag)
	}
	fmt.Fprintf(osStdout, "ok: tag format %s\n", tag)

	// 검사 2: working tree가 깨끗하다.
	out, err := runOut("git", "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("cannot check the working tree: %w", err)
	}
	if strings.TrimSpace(out) != "" {
		return fmt.Errorf("working tree is dirty (check `git status`; commit or stash first)")
	}
	fmt.Fprintln(osStdout, "ok: working tree is clean")

	// 검사 3: 현재 branch가 default branch다. default branch는 로컬 조회인
	// origin/HEAD로 구한다. symbolic-ref는 "refs/remotes/origin/<branch>" 전체를
	// 반환하므로 접두어를 벗긴다.
	const originHeadPrefix = "refs/remotes/origin/"
	ref, err := runOut("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	ref = strings.TrimSpace(ref)
	if err != nil || !strings.HasPrefix(ref, originHeadPrefix) {
		return fmt.Errorf("cannot resolve the default branch from origin/HEAD (run: git remote set-head origin --auto)")
	}
	defaultBranch := strings.TrimPrefix(ref, originHeadPrefix)
	branch, err := runOut("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || branch == "" || branch == "HEAD" {
		return fmt.Errorf("cannot resolve the current branch (detached HEAD?); the default branch is %s", defaultBranch)
	}
	if branch != defaultBranch {
		return fmt.Errorf("current branch %q is not the default branch %q", branch, defaultBranch)
	}
	fmt.Fprintf(osStdout, "ok: current branch %s (default)\n", branch)

	// 검사 4: 로컬에 같은 tag가 없다. tag 재사용·이동은 금지다(ADR 0001).
	if _, err := runOut("git", "show-ref", "--verify", "--quiet", "refs/tags/"+tag); err == nil {
		return fmt.Errorf("tag %q already exists locally; reusing or moving a released tag is not allowed (remove it only if it was never pushed)", tag)
	}
	fmt.Fprintf(osStdout, "ok: tag %s does not exist locally\n", tag)

	// 검사 5: HEAD가 원격 default branch의 현재 tip과 같다. 이 검사가 있어야
	// workflow 게이트(태그 커밋 == 원격 default tip)를 보증한다.
	headSha, err := runOut("git", "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("cannot resolve HEAD: %w", err)
	}
	remoteTip, err := lsRemoteTip(defaultBranch)
	if err != nil {
		return err
	}
	if headSha != remoteTip {
		return fmt.Errorf("HEAD %s is not the remote tip of %s (%s); pull or push first", headSha, defaultBranch, remoteTip)
	}
	fmt.Fprintf(osStdout, "ok: HEAD is the remote tip of %s\n", defaultBranch)

	// 검사 6: 원격에 같은 tag가 없다. 조회 실패는 tag 부재로 취급하지 않는다.
	if err := requireRemoteTagAbsent(tag); err != nil {
		return err
	}
	fmt.Fprintf(osStdout, "ok: tag %s does not exist on origin\n", tag)
	fmt.Fprintln(osStdout, "preflight checks passed; creating the release commit, tag, and atomic push")

	defaultRef := "HEAD:refs/heads/" + defaultBranch
	tagRef := "refs/tags/" + tag

	// 실행 1: 빈 릴리즈 커밋. 서명 정책은 Git 전달 명령과 같다(ADR 0004).
	if code := execPrepareChild("create the empty release commit", "commit created; the tag and push are not done yet",
		"commit", []string{"commit", "--no-gpg-sign", "--allow-empty", "-m", "release: " + tag},
		fmt.Sprintf("fix the failure and re-run: gg release prepare %s", tag)); code != 0 {
		return exitSilently(code)
	}

	// 실행 2: HEAD의 annotated tag.
	if code := execPrepareChild("create the annotated tag", "commit exists but the tag was not created",
		"tag", []string{"tag", "-a", tag, "-m", "Release " + tag},
		fmt.Sprintf("undo with: git reset --hard HEAD~1 (then re-run: gg release prepare %s)", tag)); code != 0 {
		return exitSilently(code)
	}

	// 실행 3: 커밋과 tag를 같은 원격 transaction으로 push한다. 원격이 atomic을
	// 지원하지 않으면 실패하며 fallback은 없다(ADR 0001).
	pushArgs := []string{"push", "--atomic", "origin", defaultRef, tagRef}
	if code := execPrepareChild("push the commit and the tag atomically", "the commit and the tag exist locally but were not pushed",
		"push", pushArgs,
		fmt.Sprintf("re-run the atomic push (%s %s) or undo locally (git reset --hard HEAD~1 && git tag -d %s) and re-run: gg release prepare %s",
			"git push --atomic origin", defaultRef+" "+tagRef, tag, tag)); code != 0 {
		return exitSilently(code)
	}

	fmt.Fprintf(osStdout, "pushed %s and %s to origin\n", defaultRef, tagRef)
	return nil
}

// execPrepareChild는 한 단계를 실행한다. 자식의 stdout·stderr·exit code는 그대로
// 보존되고(ADR 0004), 실패하면 어디까지 성공했는지와 복구 방법을 stderr 한 줄로
// 남긴다. rollback은 하지 않는다(ADR 0005).
func execPrepareChild(progress, state, name string, args []string, recovery string) int {
	fmt.Fprintf(osStdout, "%s...\n", progress)
	code := execChild(Invocation{Bin: "git", Args: args})
	if code != 0 {
		fmt.Fprintf(osStderr, "gg: %s (state: %s); recovery: %s\n", name, state, recovery)
	}
	return code
}

// exitSilently는 자식의 exit code를 그대로 반환한다. 실패 안내는 이미
// execPrepareChild가 stderr에 남겼으므로 run()은 메시지 없이 코드만 반영한다.
func exitSilently(code int) error {
	return exitCodeError{Code: code}
}

// lsRemoteTip은 원격 default branch의 현재 tip을 조회한다. 조회 자체의 실패는
// 부재로 취급하지 않고 실패한다(ADR 0005).
func lsRemoteTip(defaultBranch string) (string, error) {
	out, err := runOut("git", "ls-remote", "origin", "refs/heads/"+defaultBranch)
	if err != nil {
		return "", fmt.Errorf("cannot query the remote tip of %s (a failed lookup is not treated as absence): %w", defaultBranch, err)
	}
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return "", fmt.Errorf("the remote %s has no tip; is the branch published?", defaultBranch)
	}
	return fields[0], nil
}

// requireRemoteTagAbsent은 원격에 같은 tag가 없음을 확인한다.
func requireRemoteTagAbsent(tag string) error {
	_, err := runOut("git", "ls-remote", "--exit-code", "--tags", "origin", "refs/tags/"+tag)
	if err == nil {
		return fmt.Errorf("tag %q already exists on origin; reusing or moving a released tag is not allowed", tag)
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 2 {
		return nil // --exit-code의 2는 "일치하는 ref 없음"이다
	}
	return fmt.Errorf("cannot query the remote tag %s (a failed lookup is not treated as absence): %w", tag, err)
}

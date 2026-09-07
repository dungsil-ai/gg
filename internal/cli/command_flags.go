package cli

// flagDef는 action이 받는 flag 하나다.
type flagDef struct {
	name string                   // "--limit"
	arg  string                   // 값 placeholder(예: "<N>"). 빈 문자열이면 boolean flag다.
	desc string                   // help에 표시할 설명
	str  func(*Request) *string   // 값이 저장될 Request 필드
	bin  func(*Request) *bool     // 켤 boolean Request 필드
	list func(*Request) *[]string // 반복 지정할 때마다 값을 추가할 Request 슬라이스 필드
}

var (
	limitFlag = flagDef{name: "--limit", arg: "<N>", desc: "Limit the result count",
		str: func(r *Request) *string { return &r.Limit }}
	stateFlag = flagDef{name: "--state", arg: "<open|closed|all>", desc: "Filter by state",
		str: func(r *Request) *string { return &r.State }}
	titleFlag = flagDef{name: "--title", arg: "<text>", desc: "Set the title",
		str: func(r *Request) *string { return &r.Title }}
	bodyFlag = flagDef{name: "--body", arg: "<text>", desc: "Set the body",
		str: func(r *Request) *string { return &r.Body }}
	baseFlag = flagDef{name: "--base", arg: "<branch>", desc: "Set the base branch",
		str: func(r *Request) *string { return &r.Base }}
	headFlag = flagDef{name: "--head", arg: "<branch>", desc: "Set the head branch",
		str: func(r *Request) *string { return &r.Head }}
	draftFlag = flagDef{name: "--draft", desc: "Create a draft pull request",
		bin: func(r *Request) *bool { return &r.Draft }}
	undoFlag = flagDef{name: "--undo", desc: "Convert the pull request to a draft",
		bin: func(r *Request) *bool { return &r.Undo }}
	descriptionFlag = flagDef{name: "--description", arg: "<text>", desc: "Set the description",
		str: func(r *Request) *string { return &r.Description }}
	nameFlag = flagDef{name: "--name", arg: "<text>", desc: "Set the label name",
		str: func(r *Request) *string { return &r.Name }}
	colorFlag = flagDef{name: "--color", arg: "<hex>", desc: "Set the label color",
		str: func(r *Request) *string { return &r.Color }}
	// issue type의 --name은 종류 이름이고 label create의 --name은 label 이름이다.
	// label처럼 같은 flag 문자열을 action별 정의로 나눠 받는다.
	issueTypeNameFlag = flagDef{name: "--name", arg: "<name>", desc: "Set the issue type name",
		str: func(r *Request) *string { return &r.IssueType }}
	// label edit의 --name은 새 이름이고 고칠 label은 positional로 받는다. create의
	// --name과 같은 문자열이지만 다른 Request 필드를 채운다.
	labelEditNameFlag = flagDef{name: "--name", arg: "<text>", desc: "Set the new label name",
		str: func(r *Request) *string { return &r.NewName }}
	// issue lock의 --reason은 gh의 잠금 사유 enum(off_topic, resolved, spam,
	// too_heated)을 받는다. 값 검증은 lock action의 setPos가 한다.
	lockReasonFlag = flagDef{name: "--reason", arg: "<reason>", desc: "Lock reason (off_topic, resolved, spam, too_heated)",
		str: func(r *Request) *string { return &r.Reason }}
	parentFlag = flagDef{name: "--parent", arg: "<number>", desc: "Set the parent issue number",
		str: func(r *Request) *string { return &r.Parent }}
	blockerFlag = flagDef{name: "--blocker", arg: "<number>", desc: "Set the blocking issue number",
		str: func(r *Request) *string { return &r.Blocker }}
	publicFlag = flagDef{name: "--public", desc: "Create a public repository",
		bin: func(r *Request) *bool { return &r.Public }}
	privateFlag = flagDef{name: "--private", desc: "Create a private repository",
		bin: func(r *Request) *bool { return &r.Private }}
	// edit의 --public/--private는 create와 같은 Request 필드를 켜지만 만들기가
	// 아니라 가시성 변경이다.
	editPublicFlag = flagDef{name: "--public", desc: "Change the repository visibility to public",
		bin: func(r *Request) *bool { return &r.Public }}
	editPrivateFlag = flagDef{name: "--private", desc: "Change the repository visibility to private",
		bin: func(r *Request) *bool { return &r.Private }}
	yesFlag = flagDef{name: "--yes", desc: "Skip the confirmation prompt",
		bin: func(r *Request) *bool { return &r.Yes }}
	// sync의 --branch는 pr/issue 계열의 "filter by branch"와 뜻이 다르다.
	syncBranchFlag = flagDef{name: "--branch", arg: "<branch>", desc: "Branch to sync",
		str: func(r *Request) *string { return &r.Branch }}
	sourceFlag = flagDef{name: "--source", arg: "<repository>", desc: "Source repository to sync from",
		str: func(r *Request) *string { return &r.Source }}
	forceFlag = flagDef{name: "--force", desc: "Hard reset the destination branch to match the source",
		bin: func(r *Request) *bool { return &r.Force }}
	unsetDefaultFlag = flagDef{name: "--unset", desc: "Unset the current default repository",
		bin: func(r *Request) *bool { return &r.Unset }}
	viewDefaultFlag = flagDef{name: "--view", desc: "View the current default repository",
		bin: func(r *Request) *bool { return &r.View }}
	allowInsecureHTTPFlag = flagDef{name: "--allow-insecure-http", desc: "Allow insecure HTTP clone",
		bin: func(r *Request) *bool { return &r.AllowInsecureHTTP }}
	mergeFlag = flagDef{name: "--merge", desc: "Merge the pull request",
		bin: func(r *Request) *bool { return &r.Merge }}
	squashFlag = flagDef{name: "--squash", desc: "Squash and merge the pull request",
		bin: func(r *Request) *bool { return &r.Squash }}
	rebaseFlag = flagDef{name: "--rebase", desc: "Rebase and merge the pull request",
		bin: func(r *Request) *bool { return &r.Rebase }}
	deleteBranchFlag = flagDef{name: "--delete-branch", desc: "Delete the source branch after merging",
		bin: func(r *Request) *bool { return &r.DeleteBranch }}
	autoMergeFlag = flagDef{name: "--auto", desc: "Enable auto-merge after required approvals and CI pass",
		bin: func(r *Request) *bool { return &r.Auto }}
	// pr review의 리뷰 종류 flag. setPos에서 정확히 하나만 켜졌는지 검증한다.
	approveFlag = flagDef{name: "--approve", desc: "Approve the pull request",
		bin: func(r *Request) *bool { return &r.Approve }}
	requestChangesFlag = flagDef{name: "--request-changes", desc: "Request changes on the pull request",
		bin: func(r *Request) *bool { return &r.RequestChanges }}
	reviewCommentFlag = flagDef{name: "--comment", desc: "Leave a review comment on the pull request",
		bin: func(r *Request) *bool { return &r.ReviewComment }}
	// issue develop의 --list와 --checkout은 gh의 개발 branch 연결 표면이다.
	developListFlag = flagDef{name: "--list", desc: "List branches linked to the issue",
		bin: func(r *Request) *bool { return &r.List }}
	developCheckoutFlag = flagDef{name: "--checkout", desc: "Check out the branch after creating it",
		bin: func(r *Request) *bool { return &r.Checkout }}
	// develop의 --name은 만들 branch 이름이다.
	developNameFlag = flagDef{name: "--name", arg: "<branch>", desc: "Name of the branch to create",
		str: func(r *Request) *string { return &r.Name }}
	// label clone의 --force는 이미 있는 label을 덮어쓴다.
	labelCloneForceFlag = flagDef{name: "--force", desc: "Overwrite existing labels in the destination",
		bin: func(r *Request) *bool { return &r.Force }}
	branchFlag = flagDef{name: "--branch", arg: "<branch>", desc: "Filter by branch",
		str: func(r *Request) *string { return &r.Branch }}
	notesFlag = flagDef{name: "--notes", arg: "<text>", desc: "Set the release notes",
		str: func(r *Request) *string { return &r.Notes }}
	refFlag = flagDef{name: "--ref", arg: "<ref>", desc: "Branch or commit SHA to tag when the tag does not exist",
		str: func(r *Request) *string { return &r.Ref }}
	// release create/edit의 --draft는 pr의 --draft와 같은 Request 필드를 켜지만
	// help 문구가 다르다. action별 정의로 나눠 받는다.
	releaseDraftFlag = flagDef{name: "--draft", desc: "Save the release as a draft instead of publishing it",
		bin: func(r *Request) *bool { return &r.Draft }}
	prereleaseFlag = flagDef{name: "--prerelease", desc: "Mark the release as a prerelease",
		bin: func(r *Request) *bool { return &r.Prerelease }}
	cleanupTagFlag = flagDef{name: "--cleanup-tag", desc: "Delete the tag along with the release",
		bin: func(r *Request) *bool { return &r.CleanupTag }}
	patternFlag = flagDef{name: "--pattern", arg: "<glob>", desc: "Download only assets that match the glob",
		str: func(r *Request) *string { return &r.Pattern }}
	dirFlag = flagDef{name: "--dir", arg: "<dir>", desc: "Directory to download assets into",
		str: func(r *Request) *string { return &r.Dir }}
	filterPathFlag = flagDef{name: "--path", arg: "<path>", desc: "Limit history to this path (repeatable)",
		list: func(r *Request) *[]string { return &r.FilterPaths }}
	filterInvertPathsFlag = flagDef{name: "--invert-paths", desc: "Remove the listed paths instead of keeping them",
		bin: func(r *Request) *bool { return &r.FilterInvert }}
	filterPathRenameFlag = flagDef{name: "--path-rename", arg: "<old:new>", desc: "Rename a path prefix in history (repeatable)",
		list: func(r *Request) *[]string { return &r.FilterRenames }}
	filterReplaceTextFlag = flagDef{name: "--replace-text", arg: "<regex==>replacement|file>", desc: "Replace file contents matching regex (repeatable)",
		list: func(r *Request) *[]string { return &r.FilterReplaces }}
	filterMailmapFlag = flagDef{name: "--mailmap", arg: "<file>", desc: "Rewrite authors/committers using a mailmap file",
		str: func(r *Request) *string { return &r.FilterMailmap }}
	filterForceFlag = flagDef{name: "--force", desc: "Confirm history rewrite on a fresh clone backup",
		bin: func(r *Request) *bool { return &r.FilterForce }}
	filterDryRunFlag = flagDef{name: "--dry-run", desc: "Preview the rewrite without changing history",
		bin: func(r *Request) *bool { return &r.FilterDryRun }}
)

// 저장소 문맥과 설명 모드 flag. 파싱은 전역/flagLoop의 공통 분기가 하고,
// 정의는 help에 표시할 범위를 밝히는 데 쓴다.
var (
	repoContextFlag   = flagDef{name: "--repo", arg: "<URL>", desc: "이 URL을 저장소 문맥으로 사용"}
	remoteContextFlag = flagDef{name: "--remote", arg: "<name>", desc: "이 Git remote를 저장소 문맥으로 사용"}
	explainFlag       = flagDef{name: "--explain", desc: "선택한 저장소 문맥, Provider, 실행할 CLI를 설명"}
	helpFlag          = flagDef{name: "--help", desc: "Show help"}
)

// repo create의 --repo는 저장소 문맥이 아니라 만들 저장소 URL이다.
// 값 파싱은 flagLoop의 공통 --repo 분기가 req.RepoFlag에 채우므로, 이 flagDef는
// str/bin 세터가 없다 — help 텍스트 전용이다.
var createRepoFlag = flagDef{name: "--repo", arg: "<URL>", desc: "이 URL에 새 저장소를 만든다"}

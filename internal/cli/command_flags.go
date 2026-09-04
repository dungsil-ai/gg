package cli

// flagDef는 action이 받는 flag 하나다.
type flagDef struct {
	name string                 // "--limit"
	arg  string                 // 값 placeholder(예: "<N>"). 빈 문자열이면 boolean flag다.
	desc string                 // help에 표시할 설명
	str  func(*Request) *string // 값이 저장될 Request 필드
	bin  func(*Request) *bool   // 켤 boolean Request 필드
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
	parentFlag = flagDef{name: "--parent", arg: "<number>", desc: "Set the parent issue number",
		str: func(r *Request) *string { return &r.Parent }}
	blockerFlag = flagDef{name: "--blocker", arg: "<number>", desc: "Set the blocking issue number",
		str: func(r *Request) *string { return &r.Blocker }}
	publicFlag = flagDef{name: "--public", desc: "Create a public repository",
		bin: func(r *Request) *bool { return &r.Public }}
	privateFlag = flagDef{name: "--private", desc: "Create a private repository",
		bin: func(r *Request) *bool { return &r.Private }}
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
	yesFlag = flagDef{name: "--yes", desc: "Skip the confirmation prompt",
		bin: func(r *Request) *bool { return &r.Yes }}
	cleanupTagFlag = flagDef{name: "--cleanup-tag", desc: "Delete the tag along with the release",
		bin: func(r *Request) *bool { return &r.CleanupTag }}
	patternFlag = flagDef{name: "--pattern", arg: "<glob>", desc: "Download only assets that match the glob",
		str: func(r *Request) *string { return &r.Pattern }}
	dirFlag = flagDef{name: "--dir", arg: "<dir>", desc: "Directory to download assets into",
		str: func(r *Request) *string { return &r.Dir }}
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

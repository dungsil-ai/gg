package cli

// 이 파일은 gg가 지원하는 명령, action, flag의 유일한 정의 골격이자, cmd_repo.go /
// cmd_issue.go / cmd_pr.go / cmd_config.go / cmd_git.go가 등록한 정의를 취합하는
// registry다. ParseRequest는 이 정의로 인자를 검사하고, help 렌더링도 같은 정의를 쓴다.

// actionDef는 resource가 지원하는 action 하나다.
type actionDef struct {
	name    string
	summary string // resource help Commands 목록에 쓰는 한 줄 설명
	usage   string // action help의 Usage 줄
	flags   []flagDef

	showRepo    bool // help에 --repo 표시
	showRemote  bool // help에 --remote 표시
	showExplain bool // help에 --explain 표시
	remoteOK    bool // parser가 --remote를 허용
	explainOK   bool // parser가 --explain을 허용

	passthrough    bool                                   // flag 검사 없이 모든 인자를 git으로 전달
	minPos, maxPos int                                    // positional 개수 범위. maxPos가 -1이면 제한 없음
	posErr         string                                 // positional 개수 위반 오류. 비었으면 unexpected argument 오류
	setPos         func(req *Request, pos []string) error // positional 저장과 추가 검증
}

// resourceDef는 최상위 명령 하나다.
type resourceDef struct {
	name    string
	summary string // 최상위 help Commands 목록 설명
	desc    string // resource help 첫 줄
	usage   string // resource help Usage 줄
	actions []actionDef
}

func (rd *resourceDef) action(name string) *actionDef {
	for i := range rd.actions {
		if rd.actions[i].name == name {
			return &rd.actions[i]
		}
	}
	return nil
}

// setState/setNumber는 issue와 pr이 공유하는 positional 저장 helper다.
func setState(req *Request, pos []string) error {
	switch req.State {
	case "", "open", "closed", "all":
		return nil
	}
	return usageErr("--state must be open, closed, or all")
}

func setNumber(req *Request, pos []string) error {
	req.Number = pos[0]
	return nil
}

// commandDefs는 cmd_*.go가 등록한 resourceDef를 최상위 명령 이름으로 모은다.
var commandDefs = map[string]*resourceDef{
	"repo":   repoResourceDef,
	"issue":  issueResourceDef,
	"label":  labelResourceDef,
	"pr":     prResourceDef,
	"config": configResourceDef,
}

// commandOrder는 최상위 help의 resource 표시 순서다.
var commandOrder = []string{"repo", "issue", "label", "pr", "config"}

// invocationTable은 "<resource> <action>" 키로 gh/glab/tea의 arg-builder를 모은다.
// cmd_repo.go / cmd_issue.go / cmd_label.go / cmd_pr.go가 각자의 table을 등록하고
// 여기서 취합한다.
var invocationTable = mergeInvocationTables(repoInvocationTable, issueInvocationTable, labelInvocationTable, prInvocationTable)

func mergeInvocationTables(tables ...map[string]providerBuilders) map[string]providerBuilders {
	merged := make(map[string]providerBuilders)
	for _, table := range tables {
		for key, builders := range table {
			merged[key] = builders
		}
	}
	return merged
}

// commandAlias는 alias가 가리키는 canonical 명령 경로다.
// action이 비어 있으면 resource alias이고, 있으면 resource action alias다.
type commandAlias struct {
	resource string
	action   string
}

// commandAliases는 alias를 실제 구현이 등록된 canonical 명령 경로로 연결한다.
var commandAliases = func() map[string]commandAlias {
	aliases := map[string]commandAlias{
		"mr": {resource: "pr"},
	}
	for _, action := range commandDefs["repo"].actions {
		if _, isResource := commandDefs[action.name]; isResource {
			continue
		}
		if _, exists := aliases[action.name]; exists {
			continue
		}
		aliases[action.name] = commandAlias{resource: "repo", action: action.name}
	}
	return aliases
}()

func resolveAlias(command string) (resource, action string) {
	if canonical, ok := commandAliases[command]; ok {
		return canonical.resource, canonical.action
	}
	return command, ""
}

// helpAliases는 repo 생략 형태로 --help를 제공하는 action이다.
// --help도 git에 전달해야 하는 alias는 제외한다.
var helpAliases = map[string]bool{
	"list": true, "view": true, "create": true, "clone": true, "pull": true, "push": true,
}

package cli

import (
	"strings"
)

const topLevelHelpHead = `gg sends common Git forge commands to gh, glab, or tea, and runs supported Git commands directly.

Usage:
  gg [flags] <command>
  gg <supported-git-command> [git args]
  gg repo --help
  gg config --help
  gg issue --help
  gg issue list --help
  gg pr create --help
  gg pr status --help
  gg pr ready --help
  gg pr merge --help
  gg pr close --help
  gg pr reopen --help

Commands:
`

const topLevelHelpTail = `
Flags:
  --repo <URL>      이 URL을 저장소 문맥으로 사용
  --remote <name>   이 Git remote를 저장소 문맥으로 사용
  --explain         선택한 저장소 문맥, Provider, 실행할 CLI를 설명
  -h, --help        Show top-level help
  --version         gg 버전만 표시
  -v, -verison      단독 사용 시 gg와 설치된 git, gh, glab, tea 버전을 표시`

// topLevelHelp는 최상위 help를 명령 정의에서 만든다.
func topLevelHelp() string {
	var b strings.Builder
	b.WriteString(topLevelHelpHead)
	rows := make([][2]string, 0, len(commandOrder)+len(commandDefs["repo"].actions)+2)
	for _, name := range commandOrder {
		rows = append(rows, [2]string{name, commandDefs[name].summary})
	}
	for _, action := range commandDefs["repo"].actions {
		if action.passthrough {
			rows = append(rows, [2]string{action.name, action.summary})
		}
	}
	rows = append(rows,
		[2]string{"version", "Show gg version"},
		[2]string{"help", "Show this help"})
	b.WriteString(renderRows(rows, 4))
	b.WriteString(topLevelHelpTail)
	return b.String()
}

// renderResourceHelp는 resource 단계 help를 만든다.
func renderResourceHelp(rd *resourceDef) string {
	var b strings.Builder
	b.WriteString(rd.desc + "\n\nUsage:\n  " + rd.usage + "\n\nCommands:\n")
	rows := make([][2]string, 0, len(rd.actions))
	for i := range rd.actions {
		rows = append(rows, [2]string{rd.actions[i].name, rd.actions[i].summary})
	}
	b.WriteString(renderRows(rows, 3))
	b.WriteString("\nFlags:\n")
	b.WriteString(renderFlagLines(resourceFlags(rd)))
	return strings.TrimSuffix(b.String(), "\n")
}

// resourceFlags는 resource의 모든 action이 함께 지원하는 flag만 모은다.
func resourceFlags(rd *resourceDef) []flagDef {
	repo, remote, explain := true, true, true
	for i := range rd.actions {
		ad := &rd.actions[i]
		repo = repo && ad.showRepo
		remote = remote && ad.showRemote
		explain = explain && ad.showExplain
	}
	flags := make([]flagDef, 0, 4)
	if repo {
		flags = append(flags, repoContextFlag)
	}
	if remote {
		flags = append(flags, remoteContextFlag)
	}
	if explain {
		flags = append(flags, explainFlag)
	}
	return append(flags, helpFlag)
}

// renderActionHelp는 action 단계 help를 만든다.
func renderActionHelp(rd *resourceDef, ad *actionDef) string {
	var b strings.Builder
	b.WriteString(ad.summary + ".\n\nUsage:\n  " + ad.usage + "\n\nFlags:\n")
	b.WriteString(renderFlagLines(actionFlags(ad)))
	return strings.TrimSuffix(b.String(), "\n")
}

// actionFlags는 action 자체 flag와 문맥 flag, --help를 순서대로 모은다.
func actionFlags(ad *actionDef) []flagDef {
	flags := make([]flagDef, 0, len(ad.flags)+4)
	flags = append(flags, ad.flags...)
	if ad.showRepo {
		flags = append(flags, repoContextFlag)
	}
	if ad.showRemote {
		flags = append(flags, remoteContextFlag)
	}
	if ad.showExplain {
		flags = append(flags, explainFlag)
	}
	return append(flags, helpFlag)
}

// nestedHelp는 마지막 인자가 --help이고 나머지가 명령 경로일 때 그 경로의 help를 돌려준다.
func nestedHelp(args []string) (string, bool) {
	if len(args) < 2 || args[len(args)-1] != "--help" {
		return "", false
	}
	path := args[:len(args)-1]
	hasLeadingGlobal := false
	// gg 전역 flag는 일반 명령의 help 대상을 바꾸지 않는다. passthrough에는
	// 저장소 문맥이나 설명 모드가 없으므로 ParseRequest가 이를 검증하게 둔다.
	for len(path) >= 2 && (path[0] == "--repo" || path[0] == "--remote") {
		hasLeadingGlobal = true
		path = path[2:]
	}
	if len(path) >= 1 && path[0] == "--explain" {
		hasLeadingGlobal = true
		path = path[1:]
	}
	if len(path) == 0 {
		return "", false
	}

	head, aliasAction := resolveAlias(path[0])
	if aliasAction != "" {
		if len(path) != 1 || !helpAliases[path[0]] {
			return "", false
		}
		rd := commandDefs[head]
		ad := rd.action(aliasAction)
		if hasLeadingGlobal && ad.passthrough {
			return "", false
		}
		return renderActionHelp(rd, ad), true
	}

	switch len(path) {
	case 1:
		if rd, ok := commandDefs[head]; ok {
			return renderResourceHelp(rd), true
		}
	case 2:
		if rd, ok := commandDefs[head]; ok {
			if ad := rd.action(path[1]); ad != nil {
				if (rd.name == "repo" && isGitPassthroughAction(ad.name)) || (hasLeadingGlobal && ad.passthrough) {
					return "", false
				}
				return renderActionHelp(rd, ad), true
			}
		}
	case 3:
		// 2단어 action의 3단 경로다(gg pr comment list --help).
		if rd, ok := commandDefs[head]; ok {
			if ad := rd.action(path[1] + " " + path[2]); ad != nil {
				return renderActionHelp(rd, ad), true
			}
		}
	}
	return "", false
}

// renderRows는 "이름 설명" 행을 name 열 기준으로 맞춰 만든다.
func renderRows(rows [][2]string, gap int) string {
	width := 0
	for _, r := range rows {
		if len(r[0]) > width {
			width = len(r[0])
		}
	}
	var b strings.Builder
	for _, r := range rows {
		b.WriteString("  " + r[0] + strings.Repeat(" ", width-len(r[0])+gap) + r[1] + "\n")
	}
	return b.String()
}

// renderFlagLines는 flag 목록을 이름 열 기준으로 맞춰 만든다.
func renderFlagLines(flags []flagDef) string {
	rows := make([][2]string, 0, len(flags))
	for _, f := range flags {
		name := f.name
		if f.arg != "" {
			name += " " + f.arg
		}
		rows = append(rows, [2]string{name, f.desc})
	}
	return strings.TrimSuffix(renderRows(rows, 3), "\n")
}

// actionFlagMaps는 action 정의의 flag을 Request 필드와 연결한다.
func actionFlagMaps(ad *actionDef, req *Request) (map[string]*string, map[string]*bool) {
	strs := make(map[string]*string)
	bools := make(map[string]*bool)
	for i := range ad.flags {
		f := &ad.flags[i]
		switch {
		case f.str != nil:
			strs[f.name] = f.str(req)
		case f.bin != nil:
			bools[f.name] = f.bin(req)
		}
	}
	return strs, bools
}

// needsAction은 action이 빠졌을 때 오류 메시지를 만든다.
func needsAction(head string, rd *resourceDef) string {
	names := make([]string, 0, len(rd.actions))
	for i := range rd.actions {
		names = append(names, rd.actions[i].name)
	}
	return head + " needs an action: " + strings.Join(names, ", ")
}

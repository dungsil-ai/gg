package cli

// releaseResourceDef는 "release" 최상위 명령의 정의다: list, view, create, edit,
// delete, download, upload, delete-asset. 공통 6개 action은 gh/glab으로 중계하고,
// edit/delete-asset은 glab에 없는 하위 명령이라 glab builder를 등록하지 않아
// dispatch의 미지원 오류로 걸러진다. tea는 teaInvocation의 사전 가드에서 전체
// 미지원으로 확정한다 (ci의 tea 가드와 같은 원칙: provider별 예외는 감추지 않고
// 명시적으로 남긴다).
var releaseResourceDef = &resourceDef{
	name:    "release",
	summary: "List, view, create, edit, or delete releases, and download or upload release assets",
	desc:    "List, view, create, edit, or delete releases, and download or upload release assets.",
	usage:   "gg release <command> [flags]",
	actions: []actionDef{
		{
			name: "list", summary: "List releases", usage: "gg release list [flags]",
			flags:    []flagDef{limitFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
		},
		{
			name: "view", summary: "View one release (default: latest)", usage: "gg release view [<tag>] [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			maxPos: 1,
			posErr: "usage: gg release view [<tag>]",
			setPos: setTag,
		},
		{
			name: "create", summary: "Create a release and upload assets", usage: "gg release create <tag> [asset...] [flags]",
			flags:    []flagDef{titleFlag, notesFlag, refFlag, releaseDraftFlag, prereleaseFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: -1,
			posErr: "usage: gg release create <tag> [asset...]",
			setPos: setTagAndAssets,
		},
		{
			name: "edit", summary: "Edit a release (GitHub only)", usage: "gg release edit <tag> [flags]",
			flags:    []flagDef{titleFlag, notesFlag, releaseDraftFlag, prereleaseFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg release edit <tag>",
			setPos: setTag,
		},
		{
			name: "delete", summary: "Delete a release", usage: "gg release delete <tag> [flags]",
			flags:    []flagDef{yesFlag, cleanupTagFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 1, maxPos: 1,
			posErr: "usage: gg release delete <tag>",
			setPos: setTag,
		},
		{
			name: "download", summary: "Download release assets", usage: "gg release download [<tag>] [flags]",
			flags:    []flagDef{patternFlag, dirFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			maxPos: 1,
			posErr: "usage: gg release download [<tag>]",
			setPos: setTag,
		},
		{
			name: "upload", summary: "Upload assets to a release", usage: "gg release upload <tag> <asset>... [flags]",
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 2, maxPos: -1,
			posErr: "usage: gg release upload <tag> <asset>...",
			setPos: setTagAndAssets,
		},
		{
			name: "delete-asset", summary: "Delete a release asset (GitHub only)", usage: "gg release delete-asset <tag> <asset> [flags]",
			flags:    []flagDef{yesFlag},
			showRepo: true, showRemote: true, showExplain: true,
			remoteOK: true, explainOK: true,
			minPos: 2, maxPos: 2,
			posErr: "usage: gg release delete-asset <tag> <asset>",
			setPos: setTagAndAsset,
		},
	},
}

// setTag은 tag가 선택인 view/download와 필수인 edit/delete가 공유하는
// positional 저장 helper다. 빈 pos는 tag 생략(latest)이다.
func setTag(req *Request, pos []string) error {
	if len(pos) == 0 {
		return nil
	}
	req.Tag = pos[0]
	return nil
}

// setTagAndAssets은 create/upload처럼 tag 뒤에 asset 파일을 잇달아 받는 action의
// positional 저장 helper다.
func setTagAndAssets(req *Request, pos []string) error {
	req.Tag = pos[0]
	req.Files = pos[1:]
	if len(req.Files) == 0 {
		req.Files = nil
	}
	return nil
}

// setTagAndAsset은 delete-asset처럼 tag와 asset을 정확히 하나씩 받는 action의
// positional 저장 helper다.
func setTagAndAsset(req *Request, pos []string) error {
	req.Tag = pos[0]
	req.Asset = pos[1]
	return nil
}

var releaseListBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "list"}, c.target...)
		return appendKV(args, "--limit", c.req.Limit), nil
	},
	glab: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "list"}, c.target...)
		return appendKV(args, "--per-page", c.req.Limit), nil
	},
}

var releaseViewBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		return append(releaseViewArgs(c), c.target...), nil
	},
	glab: func(c invocationContext) (args, env []string) {
		return append(releaseViewArgs(c), c.target...), nil
	},
}

func releaseViewArgs(c invocationContext) []string {
	args := []string{c.res, "view"}
	if c.req.Tag != "" {
		args = append(args, c.req.Tag)
	}
	return args
}

var releaseCreateBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create", c.req.Tag}, c.req.Files...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--notes", c.req.Notes)
		args = appendKV(args, "--target", c.req.Ref)
		if c.req.Draft {
			args = append(args, "--draft")
		}
		if c.req.Prerelease {
			args = append(args, "--prerelease")
		}
		return append(args, c.target...), nil
	},
	glab: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "create", c.req.Tag}, c.req.Files...)
		args = appendKV(args, "--name", c.req.Title)
		args = appendKV(args, "--notes", c.req.Notes)
		args = appendKV(args, "--ref", c.req.Ref)
		return append(args, c.target...), nil
	},
}

var releaseDeleteBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		return releaseDeleteArgs(c, "--cleanup-tag"), nil
	},
	glab: func(c invocationContext) (args, env []string) {
		return releaseDeleteArgs(c, "--with-tag"), nil
	},
}

func releaseDeleteArgs(c invocationContext, cleanupFlag string) []string {
	args := append([]string{c.res, "delete", c.req.Tag}, c.target...)
	if c.req.Yes {
		args = append(args, "--yes")
	}
	if c.req.CleanupTag {
		args = append(args, cleanupFlag)
	}
	return args
}

var releaseDownloadBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		return append(releaseDownloadArgs(c, "--pattern"), c.target...), nil
	},
	glab: func(c invocationContext) (args, env []string) {
		return append(releaseDownloadArgs(c, "--asset-name"), c.target...), nil
	},
}

func releaseDownloadArgs(c invocationContext, patternFlag string) []string {
	args := []string{c.res, "download"}
	if c.req.Tag != "" {
		args = append(args, c.req.Tag)
	}
	args = appendKV(args, patternFlag, c.req.Pattern)
	args = appendKV(args, "--dir", c.req.Dir)
	return args
}

var releaseUploadBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "upload", c.req.Tag}, c.req.Files...)
		return append(args, c.target...), nil
	},
	glab: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "upload", c.req.Tag}, c.req.Files...)
		return append(args, c.target...), nil
	},
}

var releaseEditBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = append([]string{c.res, "edit", c.req.Tag}, c.target...)
		args = appendKV(args, "--title", c.req.Title)
		args = appendKV(args, "--notes", c.req.Notes)
		if c.req.Draft {
			args = append(args, "--draft")
		}
		if c.req.Prerelease {
			args = append(args, "--prerelease")
		}
		return args, nil
	},
}

var releaseDeleteAssetBuilders = providerBuilders{
	gh: func(c invocationContext) (args, env []string) {
		args = []string{c.res, "delete-asset", c.req.Tag, c.req.Asset}
		if c.req.Yes {
			args = append(args, "--yes")
		}
		return append(args, c.target...), nil
	},
}

// releaseInvocationTable은 "release <action>" 키로 gh/glab의 arg-builder를 모은다.
// edit/delete-asset은 glab builder가 없어 dispatch에서 미지원으로 걸러진다.
var releaseInvocationTable = map[string]providerBuilders{
	"release list":         releaseListBuilders,
	"release view":         releaseViewBuilders,
	"release create":       releaseCreateBuilders,
	"release edit":         releaseEditBuilders,
	"release delete":       releaseDeleteBuilders,
	"release download":     releaseDownloadBuilders,
	"release upload":       releaseUploadBuilders,
	"release delete-asset": releaseDeleteAssetBuilders,
}

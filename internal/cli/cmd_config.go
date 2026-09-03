package cli

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
)

// configResourceDef는 "config" 최상위 명령의 정의다: list, set, unset.
var configResourceDef = &resourceDef{
	name:    "config",
	summary: "Provider 설정 관리",
	desc:    "Manage Provider 설정 for self-hosted hosts.",
	usage:   "gg config <command>",
	actions: []actionDef{
		{
			name: "list", summary: "List Provider 설정", usage: "gg config list",
			posErr: "usage: gg config list",
		},
		{
			name: "set", summary: "Set the Provider for a host", usage: "gg config set <host> <gh|glab|tea>",
			posErr: "usage: gg config set <host> <provider>",
			minPos: 2, maxPos: 2,
			setPos: func(req *Request, pos []string) error {
				req.ConfigHost, req.ConfigProvider = pos[0], pos[1]
				return nil
			},
		},
		{
			name: "unset", summary: "Remove the Provider for a host", usage: "gg config unset <host>",
			posErr: "usage: gg config unset <host>",
			minPos: 1, maxPos: 1,
			setPos: func(req *Request, pos []string) error {
				req.ConfigHost = pos[0]
				return nil
			},
		},
	},
}

func runConfig(req Request) error {
	switch req.Action {
	case "list":
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}
		if len(cfg.Hosts) == 0 {
			fmt.Fprintln(os.Stdout, "No provider settings.")
			return nil
		}
		hosts := make([]string, 0, len(cfg.Hosts))
		for host := range cfg.Hosts {
			hosts = append(hosts, host)
		}
		sort.Strings(hosts)
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "HOST\tPROVIDER")
		for _, host := range hosts {
			fmt.Fprintf(w, "%s\t%s\n", host, cfg.Hosts[host])
		}
		return w.Flush()
	case "set":
		host, fixed, err := normalizeProviderSettingHost(req.ConfigHost)
		if err != nil {
			return err
		}
		if fixed {
			return usageErr(host + " is a default domain and cannot be changed")
		}
		provider, err := ParseProvider(req.ConfigProvider)
		if err != nil {
			return err
		}
		return SaveProvider(host, provider)
	case "unset":
		host, fixed, err := normalizeProviderSettingHost(req.ConfigHost)
		if err != nil {
			return err
		}
		if fixed {
			return nil
		}
		return UnsetProvider(host)
	}
	return usageErr("config does not support " + req.Action)
}

func normalizeProviderSettingHost(input string) (host string, fixed bool, err error) {
	host, err = NormalizeConfigHost(input)
	if err != nil {
		return "", false, err
	}
	_, fixed = defaultProviders[host]
	return host, fixed, nil
}

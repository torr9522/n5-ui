package templates

func Game() *Definition {
	return &Definition{
		Name:        "game",
		DisplayName: "游戏分流",
		Description: "为常见游戏平台生成默认域名分流规则。",
		Rules: []Rule{
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "steampowered.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "steamcontent.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "steamstatic.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "riotgames.com"},
		},
	}
}

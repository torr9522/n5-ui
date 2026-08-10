package templates

func Streaming() *Definition {
	return &Definition{
		Name:        "streaming",
		DisplayName: "流媒体分流",
		Description: "为常见流媒体服务生成默认域名分流规则。",
		Rules: []Rule{
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "netflix.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "nflxvideo.net"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "youtube.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "googlevideo.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "disneyplus.com"},
		},
	}
}

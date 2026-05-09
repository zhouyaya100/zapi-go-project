package routing

import "strings"

type RoutingEngine struct{ Pool *ChannelPool }

var Engine = &RoutingEngine{Pool: Pool}

func (e *RoutingEngine) ResolveModel(ch *ChannelInfo, model string) string {
	actualModel := model
	// If this channel belongs to any upstream group and the requested model
	// (the group's alias) is not in the channel's model list, map alias →
	// the channel's first available model so upstream receives a real model name.
	if len(ch.UpstreamGroupIDs) > 0 {
		found := false
		for _, m := range ch.Models {
			if m == model { found = true; break }
		}
		if !found && len(ch.Models) > 0 {
			actualModel = ch.Models[0]
		}
	}
	if ch.ModelMapping != nil { if mapped, ok := ch.ModelMapping[actualModel]; ok { return mapped } }
	return actualModel
}

func (e *RoutingEngine) BuildUpstreamURL(baseURL, requestPath string) string {
	base := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(base, "/v1") { return base[:len(base)-3] + requestPath }
	return base + requestPath
}

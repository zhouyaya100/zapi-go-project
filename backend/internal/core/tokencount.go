package core

import (
	tiktoken "github.com/pkoukk/tiktoken-go"
)

var tke *tiktoken.Tiktoken

func init() {
	// cl100k_base: GPT-4 / GPT-3.5-turbo / Claude 通用编码
	enc, err := tiktoken.EncodingForModel("gpt-4")
	if err != nil {
		// Fallback: use cl100k_base directly
		enc, err = tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			// If even that fails, tke stays nil and we use rune estimation
			return
		}
	}
	tke = enc
}

// CountTokens counts tokens using tiktoken if available, otherwise estimates
func CountTokens(text string) int {
	if tke != nil {
		return len(tke.Encode(text, nil, nil))
	}
	// Fallback: rune-based estimation
	chinese := 0
	other := 0
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			chinese++
		} else {
			other++
		}
	}
	result := float64(chinese)/1.5 + float64(other)/4
	if result < 1 && len(text) > 0 {
		return 1
	}
	return int(result)
}

// CountPromptTokens counts prompt tokens from request body
func CountPromptTokens(bodyJSON map[string]interface{}) int64 {
	if bodyJSON == nil {
		return 0
	}
	total := 0
	if messages, ok := bodyJSON["messages"].([]interface{}); ok {
		for _, m := range messages {
			if msg, ok := m.(map[string]interface{}); ok {
				total += 3 // message overhead
				// Content can be string or list (multimodal)
				if content, ok := msg["content"].(string); ok {
					total += CountTokens(content)
				} else if contentList, ok := msg["content"].([]interface{}); ok {
					// Multimodal: [{"type":"text","text":"..."}, {"type":"image_url",...}]
					for _, part := range contentList {
						if partMap, ok := part.(map[string]interface{}); ok {
							if text, ok := partMap["text"].(string); ok {
								total += CountTokens(text)
							}
						}
					}
				}
				if name, ok := msg["name"].(string); ok && name != "" {
					total += CountTokens(name) + 1
				}
			}
		}
		total += 3 // reply primer
	}
	return int64(total)
}

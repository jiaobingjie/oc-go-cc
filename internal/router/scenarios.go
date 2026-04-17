package router

import (
	"fmt"
	"strings"

	"oc-go-cc/internal/config"
)

// Scenario represents the routing scenario for model selection.
type Scenario string

const (
	ScenarioDefault     Scenario = "default"
	ScenarioBackground  Scenario = "background"
	ScenarioThink       Scenario = "think"
	ScenarioComplex     Scenario = "complex"
	ScenarioLongContext Scenario = "long_context"
	ScenarioFast        Scenario = "fast"
)

// ScenarioResult contains the detected scenario and token count.
type ScenarioResult struct {
	Scenario   Scenario
	TokenCount int
	Reason     string
}

// MessageContent represents a single message in a conversation.
type MessageContent struct {
	Role    string
	Content string
}

// DetectScenario analyzes a request to determine which model to use.
// Routing priority:
//  1. Long Context (> threshold)
//  2. Complex (architectural patterns)
//  3. Think (reasoning patterns)
//  4. Background (simple operations)
//  5. Default
func DetectScenario(messages []MessageContent, tokenCount int, cfg *config.Config) ScenarioResult {
	// 1. Check for long context first (most important)
	threshold := getLongContextThreshold(cfg)
	if tokenCount > threshold {
		return ScenarioResult{
			Scenario:   ScenarioLongContext,
			TokenCount: tokenCount,
			Reason:     fmt.Sprintf("token count %d exceeds threshold %d (use MiniMax for 1M context)", tokenCount, threshold),
		}
	}

	// 2. Check for complex architectural tasks
	if hasComplexPattern(messages) {
		return ScenarioResult{
			Scenario:   ScenarioComplex,
			TokenCount: tokenCount,
			Reason:     "complex architectural pattern detected (use GLM-5.1)",
		}
	}

	// 3. Check for thinking/reasoning patterns
	if hasThinkingPattern(messages) {
		return ScenarioResult{
			Scenario:   ScenarioThink,
			TokenCount: tokenCount,
			Reason:     "thinking/reasoning pattern detected (use GLM-5)",
		}
	}

	// 4. Check for background task patterns
	if hasBackgroundPattern(messages) {
		return ScenarioResult{
			Scenario:   ScenarioBackground,
			TokenCount: tokenCount,
			Reason:     "background task pattern detected (use Qwen3.5 Plus)",
		}
	}

	// 5. Default
	return ScenarioResult{
		Scenario:   ScenarioDefault,
		TokenCount: tokenCount,
		Reason:     "default scenario (use Kimi K2.5)",
	}
}

// hasComplexPattern looks for complex architectural tasks that need GLM-5.1.
func hasComplexPattern(messages []MessageContent) bool {
	complexKeywords := []string{
		"architect", "architecture", "refactor", "redesign",
		"complex", "difficult", "challenging",
		"optimize", "performance", "efficiency",
		"design pattern", "best practice",
	}

	for _, msg := range messages {
		if msg.Role == "system" {
			lower := strings.ToLower(msg.Content)
			for _, kw := range complexKeywords {
				if strings.Contains(lower, kw) {
					return true
				}
			}
		}
	}
	return false
}

// hasThinkingPattern looks for system prompts mentioning reasoning keywords
// or content containing thinking/reasoning markers.
func hasThinkingPattern(messages []MessageContent) bool {
	thinkingKeywords := []string{
		"think", "thinking", "plan", "reason", "reasoning",
		"analyze", "analysis", "step by step",
	}

	for _, msg := range messages {
		if msg.Role == "system" {
			lower := strings.ToLower(msg.Content)
			for _, kw := range thinkingKeywords {
				if strings.Contains(lower, kw) {
					return true
				}
			}
		}
		// Check for thinking content blocks
		if strings.Contains(msg.Content, "antThinking") {
			return true
		}
	}
	return false
}

// hasBackgroundPattern looks for patterns that suggest background tasks
// such as file reading, directory listing, grep operations, or simple commands.
func hasBackgroundPattern(messages []MessageContent) bool {
	backgroundKeywords := []string{
		// File operations
		"read file", "view file", "show file", "cat file",
		"list directory", "ls -", "dir listing",
		// Search operations
		"grep", "search", "find pattern",
		// Simple info
		"what is", "what's", "tell me about",
		// Quick checks
		"check if", "verify that", "validate",
	}

	for _, msg := range messages {
		lower := strings.ToLower(msg.Content)
		for _, kw := range backgroundKeywords {
			if strings.Contains(lower, kw) {
				return true
			}
		}
	}
	return false
}

// getLongContextThreshold returns the configured threshold or a sensible default.
// Default is 60K tokens to trigger MiniMax (1M context) vs regular models (128-256K).
func getLongContextThreshold(cfg *config.Config) int {
	if lc, ok := cfg.Models["long_context"]; ok && lc.ContextThreshold > 0 {
		return lc.ContextThreshold
	}
	return 60000 // Default: 60K tokens
}

package server

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"aster/internal/ai"
	"aster/internal/builtin_tools"
	"aster/internal/react"
	"aster/internal/server/provider"
	"aster/internal/server/ws"
	"aster/internal/service"
	"aster/internal/store"
)

// AgentRunner bridges inbound WebSocket chat messages to the ReAct agent engine
// and streams the agent's output events back to connected clients. It is the
// server-side counterpart of the TUI's AgentExecContext.
type AgentRunner struct {
	hub          *ws.Hub
	db           *store.DB
	skillService *service.SkillService
	providerCfg  *provider.Config

	// Shared, immutable building blocks assembled once at construction.
	aiClient      ai.ChatClient
	clientFactory ai.ClientFactory
	registry      *react.ToolRegistry

	agentName   string
	instruction string
	msgSeq      atomic.Uint64
}

// NewAgentRunner constructs the runner and assembles the shared agent building
// blocks from the resolved provider config. Returns an error if the model
// provider is not configured — the caller decides whether to run in degraded
// (no-chat) mode.
func NewAgentRunner(hub *ws.Hub, db *store.DB, skillService *service.SkillService, cfg *provider.Config) (*AgentRunner, error) {
	if cfg == nil || !cfg.Configured() {
		return nil, fmt.Errorf("model provider not configured")
	}

	aiClient := cfg.BuildClient()
	clientFactory := ai.NewSimpleClientFactory(aiClient, func(modelID string) ai.ChatClient {
		return cfg.BuildClient()
	})

	registry := react.NewDefaultToolRegistry()
	registry.Register(builtin_tools.ListSkillsToolName, func(_ builtin_tools.ToolContext) react.Tool {
		return builtin_tools.NewListSkillsTool(skillService)
	})

	return &AgentRunner{
		hub:           hub,
		db:            db,
		skillService:  skillService,
		providerCfg:   cfg,
		aiClient:      aiClient,
		clientFactory: clientFactory,
		registry:      registry,
		agentName:     "aster",
		instruction:   defaultAgentInstruction,
	}, nil
}

// buildFactory assembles a fresh AgentFactory whose emitter forwards events to
// the given session's WebSocket clients. Building is cheap; a per-turn factory
// gives us session-scoped streaming without shared mutable state.
func (r *AgentRunner) buildFactory(sessionID string) *react.AgentFactory {
	return react.NewAgentFactory(
		react.WithFactoryDefaultAIClient(r.aiClient),
		react.WithFactoryAIClientFactory(r.clientFactory),
		react.WithFactoryToolRegistry(r.registry),
		react.WithFactorySkillsCatalog(r.skillService),
		react.WithFactorySkillLookup(skillServiceLookup{svc: r.skillService}),
		react.WithFactoryEmitterFunc(func(e *react.AgentOutputEvent) error {
			r.forwardEvent(sessionID, e)
			return nil
		}),
	)
}

// HandleChat is the ws.ChatHandler implementation. It runs one agent turn for
// the given session and streams events back to the client.
func (r *AgentRunner) HandleChat(sessionID, userID, content string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Persist the user's message first so history survives a refresh.
	if _, err := r.db.CreateMessage(sessionID, "user", content); err != nil {
		log.Printf("[agent] persist user message: %v", err)
	}

	history := r.loadHistory(sessionID)

	factory := r.buildFactory(sessionID)
	agent, err := factory.Build(react.AgentDefinition{
		Name:        r.agentName,
		Instruction: r.instruction,
		ModelID:     r.providerCfg.Model,
		Policies: react.AgentPolicies{
			MaxIterations: 40,
		},
	})
	if err != nil {
		r.sendError(sessionID, fmt.Sprintf("build agent: %v", err))
		return
	}
	if len(history) > 0 {
		agent.SetHistory(ai.NormalizeMsgInfoSlice(history))
	}

	result, execErr := agent.Execute(ctx, content)

	finalText := ""
	if result != nil {
		finalText = strings.TrimSpace(result.Result)
	}
	if execErr != nil {
		r.sendError(sessionID, execErr.Error())
	}

	if finalText != "" {
		if _, err := r.db.CreateMessage(sessionID, "assistant", finalText); err != nil {
			log.Printf("[agent] persist assistant message: %v", err)
		}
		// Emit a final assistant message in the frontend's Message shape so the
		// chat renders a clean bubble regardless of streaming granularity.
		r.hub.SendEvent(sessionID, ws.StreamEvent{
			Type: "message",
			Payload: map[string]any{
				"id":        r.nextMessageID(),
				"role":      "assistant",
				"content":   finalText,
				"timestamp": time.Now().Format(time.RFC3339),
			},
		})
	}

	r.hub.SendEvent(sessionID, ws.StreamEvent{Type: "done", Payload: map[string]any{}})
}

// loadHistory reads persisted messages and converts them to ai.MsgInfo.
func (r *AgentRunner) loadHistory(sessionID string) []*ai.MsgInfo {
	rows, err := r.db.ListMessagesBySession(sessionID)
	if err != nil {
		log.Printf("[agent] load history: %v", err)
		return nil
	}
	out := make([]*ai.MsgInfo, 0, len(rows))
	for _, m := range rows {
		switch m.Role {
		case "user":
			out = append(out, ai.NewUserMsgInfo(m.Content))
		case "assistant":
			out = append(out, ai.NewAIMsgInfo(m.Content))
		case "system":
			out = append(out, ai.NewSystemMsgInfo(m.Content))
		}
	}
	return out
}

// forwardEvent maps a react.AgentOutputEvent to a ws.StreamEvent.
func (r *AgentRunner) forwardEvent(sessionID string, e *react.AgentOutputEvent) {
	if e == nil {
		return
	}
	payload := map[string]any{
		"agent":     e.AgentName,
		"iteration": e.Iteration,
	}
	if e.Content != "" {
		payload["content"] = e.Content
	}
	if len(e.Payload) > 0 {
		payload["data"] = e.Payload
	}
	r.hub.SendEvent(sessionID, ws.StreamEvent{
		Type:    string(e.Type),
		Payload: payload,
	})
}

func (r *AgentRunner) sendError(sessionID, msg string) {
	r.hub.SendEvent(sessionID, ws.StreamEvent{
		Type:    "error",
		Payload: map[string]any{"message": msg},
	})
}

func (r *AgentRunner) nextMessageID() string {
	return fmt.Sprintf("srv-%d-%d", time.Now().UnixNano(), r.msgSeq.Add(1))
}

// skillServiceLookup adapts *service.SkillService to react.SkillLookup.
type skillServiceLookup struct {
	svc *service.SkillService
}

func (l skillServiceLookup) LookupSkill(ctx context.Context, name string) (*react.SkillInfo, error) {
	skills, err := l.svc.LoadSkills(ctx, []string{name})
	if err != nil {
		return nil, err
	}
	if len(skills) == 0 || skills[0] == nil {
		return nil, fmt.Errorf("skill %q not found", name)
	}
	s := skills[0]
	return &react.SkillInfo{
		Name:          s.Name,
		Description:   s.Description,
		Instructions:  s.Instructions,
		Agent:         s.Agent,
		WhenToUse:     s.WhenToUse,
		Arguments:     s.Arguments,
		AllowedTools:  s.AllowedTools,
		MCP:           s.MCP,
		Context:       s.Context,
		SkillDir:      s.SkillDir,
		UserInvocable: s.UserInvocable,
	}, nil
}

const defaultAgentInstruction = `You are Aster, an AI-powered security analysis agent. You help authorized security engineers with penetration testing, code auditing, vulnerability analysis, and security research. Provide precise, technically accurate guidance. When asked to analyze targets, assume the user has proper authorization.`

package session

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/domain"
)

// SessionStore persists Session lifecycle state and provides read access
// to team/member specs needed for plan building.
type SessionStore interface {
	// Session CRUD
	CreateSession(ctx context.Context, session *domain.Session) error
	GetSession(ctx context.Context, id string) (domain.Session, error)
	UpdateSessionStatus(ctx context.Context, session *domain.Session) error
	CreateMessage(ctx context.Context, msg *domain.Message) error
	CreateLog(ctx context.Context, l *domain.Log) error

	// Team and member spec reads (used by plan building)
	GetTeam(ctx context.Context, id string) (domain.Team, error)
	GetMember(ctx context.Context, id string) (domain.Member, error)
	GetCliProfile(ctx context.Context, id string) (domain.CliProfile, error)
	GetSystemPrompt(ctx context.Context, id string) (domain.SystemPrompt, error)
	GetEnv(ctx context.Context, id string) (domain.Env, error)
	GetGitRepo(ctx context.Context, id string) (domain.GitRepo, error)
}

// Terminal launches and terminates member processes.
type Terminal interface {
	Launch(sessionID, sessionName string, members []domain.MemberPlan) error
	Terminate(sessionID string) error
	Send(sessionID, teamMemberID, text string) error
	Attach(sessionID string, memberID *string) error
}

// Workspace prepares and cleans up member directories.
type Workspace interface {
	Prepare(ctx context.Context, members []domain.MemberPlan) error
	Cleanup(sessionID string) error
}

// AuthChecker reads authentication credentials for the CLI agent.
type AuthChecker interface {
	ReadToken() (string, error)
}

// Service orchestrates session execution: build plan, prepare workspace,
// launch terminals, deliver messages.
type Service struct {
	store     SessionStore
	terminal  Terminal
	workspace Workspace
	base      string
	homeDir   string
}

// New creates a session Service.
func New(store SessionStore, term Terminal, ws Workspace, base, homeDir string) *Service {
	return &Service{store: store, terminal: term, workspace: ws, base: base, homeDir: homeDir}
}

// Start builds a fresh plan from the team's current state, resolves placeholders,
// prepares the workspace, and launches terminals for each member.
func (s *Service) Start(ctx context.Context, team domain.Team, auth AuthChecker) (*domain.Session, error) {
	plan, err := s.buildPlan(ctx, team)
	if err != nil {
		return nil, fmt.Errorf("build plan: %w", err)
	}

	claudeToken := resolveAuth(auth)

	sessionID := uuid.NewString()
	members := make([]domain.MemberPlan, 0, len(plan))
	for _, m := range plan {
		members = append(members, resolvePlaceholders(m, s.base, s.homeDir, sessionID, claudeToken))
	}

	session, err := domain.NewSession(sessionID, team.ID)
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}
	session.Plan = plan

	success := false
	defer func() {
		if !success {
			_ = s.workspace.Cleanup(sessionID)
		}
	}()

	if err := s.workspace.Prepare(ctx, members); err != nil {
		return nil, fmt.Errorf("prepare workspace: %w", err)
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}

	if err := s.terminal.Launch(sessionID, team.Name, members); err != nil {
		return nil, fmt.Errorf("launch terminal: %w", err)
	}

	success = true
	return session, nil
}

// Stop terminates a running execution, cleans up workspace, and updates status.
func (s *Service) Stop(ctx context.Context, sessionID string) error {
	session, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	if err := s.terminal.Terminate(sessionID); err != nil {
		return fmt.Errorf("terminate terminal: %w", err)
	}

	if err := s.workspace.Cleanup(sessionID); err != nil {
		return fmt.Errorf("cleanup workspace: %w", err)
	}

	session.Stop()
	if err := s.store.UpdateSessionStatus(ctx, &session); err != nil {
		return fmt.Errorf("update session status: %w", err)
	}

	return nil
}

// Send delivers a message to the recipient's terminal, then persists it.
// Delivery happens first so that a bad recipient fails before anything is saved.
func (s *Service) Send(ctx context.Context, sessionID, fromTeamMemberID, toTeamMemberID, content string) error {
	session, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	text := content
	if fromTeamMemberID != "" {
		senderName := fromTeamMemberID
		for _, m := range session.Plan {
			if m.TeamMemberID == fromTeamMemberID {
				senderName = m.MemberName
				break
			}
		}
		text = fmt.Sprintf("[Message from %s] %s", senderName, content)
	}

	if err := s.terminal.Send(sessionID, toTeamMemberID, text); err != nil {
		return fmt.Errorf("deliver message: %w", err)
	}

	msg, err := domain.NewMessage(sessionID, fromTeamMemberID, toTeamMemberID, content)
	if err != nil {
		return fmt.Errorf("new message: %w", err)
	}
	if err := s.store.CreateMessage(ctx, msg); err != nil {
		return fmt.Errorf("save message: %w", err)
	}
	return nil
}

// Log persists a self-recorded entry by a team member.
func (s *Service) Log(ctx context.Context, sessionID, teamMemberID, content string) error {
	if _, err := s.store.GetSession(ctx, sessionID); err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	l, err := domain.NewLog(sessionID, teamMemberID, content)
	if err != nil {
		return fmt.Errorf("new log: %w", err)
	}
	if err := s.store.CreateLog(ctx, l); err != nil {
		return fmt.Errorf("save log: %w", err)
	}
	return nil
}

// resolveAuth reads the Claude auth token.
func resolveAuth(auth AuthChecker) string {
	token, err := auth.ReadToken()
	if err == nil && token != "" {
		return token
	}
	return ""
}

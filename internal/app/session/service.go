package session

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/domain"
)

// SessionStore persists Session lifecycle state.
type SessionStore interface {
	CreateSession(ctx context.Context, session *domain.Session) error
	GetSession(ctx context.Context, id string) (domain.Session, error)
	UpdateSessionStatus(ctx context.Context, session *domain.Session) error
	GetTeam(ctx context.Context, id string) (domain.Team, error)
	CreateMessage(ctx context.Context, msg *domain.Message) error
}

// Terminal launches and terminates member processes.
type Terminal interface {
	Launch(sessionID, sessionName string, members []domain.MemberSessionPlan) error
	Terminate(sessionID string) error
	Send(sessionID, memberID, text string) error
}

// Workspace prepares and cleans up member directories.
type Workspace interface {
	Prepare(ctx context.Context, members []domain.MemberSessionPlan) error
	Cleanup(teamID string) error
}

// AuthChecker reads authentication credentials for CLI binaries.
type AuthChecker interface {
	Check(binary domain.CliBinary) error
	ReadToken(binary domain.CliBinary) (string, error)
	ReadAuthFile(binary domain.CliBinary) ([]byte, error)
}

// Service orchestrates session execution: prepare workspace, launch terminals, deliver messages.
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

// Start resolves placeholders from the team's plan, prepares the workspace,
// and launches terminals for each member.
func (s *Service) Start(ctx context.Context, team domain.Team, auth AuthChecker) (*domain.Session, error) {
	claudeToken, codexAuth := resolveAuth(auth)

	sessionID := uuid.NewString()
	members := make([]domain.MemberSessionPlan, 0, len(team.Plan))
	for _, m := range team.Plan {
		members = append(members, resolvePlaceholders(m, s.base, s.homeDir, sessionID, claudeToken, string(codexAuth)))
	}

	session, err := domain.NewSession(sessionID, team.ID)
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}

	success := false
	defer func() {
		if !success {
			_ = s.workspace.Cleanup(team.ID)
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

	if err := s.workspace.Cleanup(session.TeamID); err != nil {
		return fmt.Errorf("cleanup workspace: %w", err)
	}

	session.Stop()
	if err := s.store.UpdateSessionStatus(ctx, &session); err != nil {
		return fmt.Errorf("update session status: %w", err)
	}

	return nil
}

// Send validates the relation, persists the message, and delivers it to the recipient's terminal.
func (s *Service) Send(ctx context.Context, sessionID, fromMemberID, toMemberID, content string) error {
	session, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	team, err := s.store.GetTeam(ctx, session.TeamID)
	if err != nil {
		return fmt.Errorf("get team: %w", err)
	}

	if err := validateDelivery(team, fromMemberID, toMemberID); err != nil {
		return err
	}

	senderName := "user"
	for _, m := range team.Plan {
		if m.MemberID == fromMemberID {
			senderName = m.MemberName
			break
		}
	}

	msg, err := domain.NewMessage(sessionID, fromMemberID, toMemberID, content)
	if err != nil {
		return fmt.Errorf("new message: %w", err)
	}
	if err := s.store.CreateMessage(ctx, msg); err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	text := fmt.Sprintf("[Message from %s] %s", senderName, content)
	return s.terminal.Send(sessionID, toMemberID, text)
}

// resolveAuth tries to read auth credentials for all known CLI binaries.
func resolveAuth(auth AuthChecker) (string, []byte) {
	var claudeToken string
	var codexAuth []byte

	if err := auth.Check(domain.BinaryClaude); err == nil {
		token, err := auth.ReadToken(domain.BinaryClaude)
		if err == nil && token != "" {
			claudeToken = token
		}
	}

	if err := auth.Check(domain.BinaryCodex); err == nil {
		data, err := auth.ReadAuthFile(domain.BinaryCodex)
		if err == nil && data != nil {
			codexAuth = data
		}
	}

	return claudeToken, codexAuth
}

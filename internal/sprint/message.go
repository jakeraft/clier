package sprint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jakeraft/clier/internal/db/generated"
	"github.com/jakeraft/clier/internal/domain"
	"github.com/jakeraft/clier/internal/terminal"
)

const surfacesFileName = "surfaces.json"

// SurfaceMap holds workspace and surface refs for a sprint.
type SurfaceMap struct {
	WorkspaceRef string            `json:"workspace_ref"`
	Surfaces     map[string]string `json:"surfaces"` // memberID → surface ref
}

func saveSurfaces(sprintsDir, sprintID string, members []domain.MemberSnapshot, result *terminal.LaunchResult) error {
	surfaces := make(map[string]string, len(members))
	for i, m := range members {
		surfaces[m.MemberID] = result.Surfaces[i]
	}
	m := SurfaceMap{
		WorkspaceRef: result.WorkspaceRef,
		Surfaces:     surfaces,
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal surfaces: %w", err)
	}
	path := filepath.Join(sprintsDir, sprintID, surfacesFileName)
	return os.WriteFile(path, data, 0644)
}

func loadSurfaces(sprintsDir, sprintID string) (*SurfaceMap, error) {
	path := filepath.Join(sprintsDir, sprintID, surfacesFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read surfaces: %w", err)
	}
	var m SurfaceMap
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse surfaces: %w", err)
	}
	return &m, nil
}

// DeliverMessage validates the relation, persists the message, and delivers it to the recipient's terminal.
func (e *Engine) DeliverMessage(ctx context.Context, sprintID, fromMemberID, toMemberID, content string) error {
	// Load sprint and validate state
	sprintRow, err := e.store.GetSprint(ctx, sprintID)
	if err != nil {
		return fmt.Errorf("get sprint: %w", err)
	}
	if domain.SprintState(sprintRow.State) != domain.SprintRunning {
		return fmt.Errorf("sprint is not running (state: %s)", sprintRow.State)
	}

	// Parse snapshot for relation validation and sender name
	var snapshot domain.TeamSnapshot
	if err := json.Unmarshal([]byte(sprintRow.TeamSnapshot), &snapshot); err != nil {
		return fmt.Errorf("parse snapshot: %w", err)
	}

	fromName, err := validateMessageRoute(snapshot, fromMemberID, toMemberID)
	if err != nil {
		return err
	}

	// Persist message
	now := time.Now().Unix()
	if err := e.store.CreateMessage(ctx, generated.CreateMessageParams{
		ID:           uuid.NewString(),
		SprintID:     sprintID,
		FromMemberID: fromMemberID,
		ToMemberID:   toMemberID,
		Content:      content,
		CreatedAt:    now,
	}); err != nil {
		return fmt.Errorf("save message: %w", err)
	}

	// Load surface ref and deliver
	surfaces, err := loadSurfaces(e.settings.SprintsDir(), sprintID)
	if err != nil {
		return fmt.Errorf("load surfaces: %w", err)
	}

	surfaceRef, ok := surfaces.Surfaces[toMemberID]
	if !ok {
		return fmt.Errorf("surface not found for member: %s", toMemberID)
	}

	text := fmt.Sprintf("[Message from %s] %s", fromName, content)
	return e.terminal.Send(surfaceRef, text)
}

func validateMessageRoute(snapshot domain.TeamSnapshot, fromID, toID string) (string, error) {
	var fromName string
	var fromRelations domain.MemberRelations
	found := false

	for _, m := range snapshot.Members {
		if m.MemberID == fromID {
			fromName = m.MemberName
			fromRelations = m.Relations
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("sender not found: %s", fromID)
	}

	// Check recipient exists
	recipientExists := false
	for _, m := range snapshot.Members {
		if m.MemberID == toID {
			recipientExists = true
			break
		}
	}
	if !recipientExists {
		return "", fmt.Errorf("recipient not found: %s", toID)
	}

	// Validate relation: sender can message leaders, workers, or peers
	if slices.Contains(fromRelations.Leaders, toID) ||
		slices.Contains(fromRelations.Workers, toID) ||
		slices.Contains(fromRelations.Peers, toID) {
		return fromName, nil
	}

	return "", fmt.Errorf("no relation from %s to %s", fromID, toID)
}

package sprint

import (
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestBuildContext(t *testing.T) {
	t.Run("RootWithWorker", func(t *testing.T) {
		snapshot := domain.TeamSnapshot{
			TeamName:     "MyTeam",
			RootMemberID: "boss-1",
			Members: []domain.MemberSnapshot{
				{
					MemberID:   "boss-1",
					MemberName: "Boss",
					Relations:  domain.MemberRelations{Workers: []string{"worker-1"}},
				},
				{
					MemberID:   "worker-1",
					MemberName: "Writer",
					Relations:  domain.MemberRelations{Leaders: []string{"boss-1"}},
				},
			},
		}

		ctx, err := BuildContext(snapshot, "sprint-1", "boss-1")
		if err != nil {
			t.Fatal(err)
		}

		if ctx.SprintID != "sprint-1" {
			t.Errorf("expected SprintID=sprint-1, got %s", ctx.SprintID)
		}
		if ctx.TeamName != "MyTeam" {
			t.Errorf("expected TeamName=MyTeam, got %s", ctx.TeamName)
		}
		if ctx.Me.MemberID != "boss-1" || ctx.Me.MemberName != "Boss" {
			t.Errorf("unexpected Me: %+v", ctx.Me)
		}
		if len(ctx.Workers) != 1 || ctx.Workers[0].MemberName != "Writer" {
			t.Errorf("unexpected Workers: %+v", ctx.Workers)
		}
		if len(ctx.Leaders) != 0 {
			t.Errorf("root should have no leaders: %+v", ctx.Leaders)
		}
	})

	t.Run("NonRootWithLeaderAndPeer", func(t *testing.T) {
		snapshot := domain.TeamSnapshot{
			TeamName:     "MyTeam",
			RootMemberID: "leader-1",
			Members: []domain.MemberSnapshot{
				{MemberID: "leader-1", MemberName: "Editor"},
				{
					MemberID:   "writer-1",
					MemberName: "Writer",
					Relations: domain.MemberRelations{
						Leaders: []string{"leader-1"},
						Peers:   []string{"peer-1"},
					},
				},
				{MemberID: "peer-1", MemberName: "Reviewer"},
			},
		}

		ctx, err := BuildContext(snapshot, "sprint-1", "writer-1")
		if err != nil {
			t.Fatal(err)
		}

		if ctx.SprintID != "sprint-1" {
			t.Errorf("expected SprintID=sprint-1, got %s", ctx.SprintID)
		}
		if len(ctx.Leaders) != 1 || ctx.Leaders[0].MemberName != "Editor" {
			t.Errorf("unexpected Leaders: %+v", ctx.Leaders)
		}
		if len(ctx.Peers) != 1 || ctx.Peers[0].MemberName != "Reviewer" {
			t.Errorf("unexpected Peers: %+v", ctx.Peers)
		}
	})

	t.Run("UserMemberID_ReturnsAllMembers", func(t *testing.T) {
		snapshot := domain.TeamSnapshot{
			TeamName:     "MyTeam",
			RootMemberID: "m-1",
			Members: []domain.MemberSnapshot{
				{MemberID: "m-1", MemberName: "Agent1"},
				{MemberID: "m-2", MemberName: "Agent2"},
			},
		}

		ctx, err := BuildContext(snapshot, "sprint-1", domain.UserMemberID)
		if err != nil {
			t.Fatal(err)
		}

		if ctx.SprintID != "sprint-1" {
			t.Errorf("expected SprintID=sprint-1, got %s", ctx.SprintID)
		}
		if ctx.Me.MemberID != domain.UserMemberID {
			t.Errorf("expected UserMemberID, got %s", ctx.Me.MemberID)
		}
		if ctx.Me.MemberName != "user" {
			t.Errorf("expected name=user, got %s", ctx.Me.MemberName)
		}
		if len(ctx.Workers) != len(snapshot.Members) {
			t.Errorf("user should see all members as workers: %+v", ctx.Workers)
		}
	})

	t.Run("UnknownMemberID_ReturnsError", func(t *testing.T) {
		snapshot := domain.TeamSnapshot{
			TeamName:     "MyTeam",
			RootMemberID: "m-1",
			Members: []domain.MemberSnapshot{
				{MemberID: "m-1", MemberName: "Agent"},
			},
		}

		_, err := BuildContext(snapshot, "sprint-1", "nonexistent")
		if err == nil {
			t.Error("should return error for unknown member ID")
		}
	})
}

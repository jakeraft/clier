package sprint

import (
	"testing"

	"github.com/jakeraft/clier/internal/domain"
)

func TestBuildPosition(t *testing.T) {
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

		pos, err := BuildPosition(snapshot, "sprint-1", "boss-1")
		if err != nil {
			t.Fatal(err)
		}

		if pos.SprintID != "sprint-1" {
			t.Errorf("expected SprintID=sprint-1, got %s", pos.SprintID)
		}
		if pos.TeamName != "MyTeam" {
			t.Errorf("expected TeamName=MyTeam, got %s", pos.TeamName)
		}
		if pos.Me.MemberID != "boss-1" || pos.Me.MemberName != "Boss" {
			t.Errorf("unexpected Me: %+v", pos.Me)
		}
		if len(pos.Workers) != 1 || pos.Workers[0].MemberName != "Writer" {
			t.Errorf("unexpected Workers: %+v", pos.Workers)
		}
		if len(pos.Leaders) != 0 {
			t.Errorf("root should have no leaders: %+v", pos.Leaders)
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

		pos, err := BuildPosition(snapshot, "sprint-1", "writer-1")
		if err != nil {
			t.Fatal(err)
		}

		if pos.SprintID != "sprint-1" {
			t.Errorf("expected SprintID=sprint-1, got %s", pos.SprintID)
		}
		if len(pos.Leaders) != 1 || pos.Leaders[0].MemberName != "Editor" {
			t.Errorf("unexpected Leaders: %+v", pos.Leaders)
		}
		if len(pos.Peers) != 1 || pos.Peers[0].MemberName != "Reviewer" {
			t.Errorf("unexpected Peers: %+v", pos.Peers)
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

		pos, err := BuildPosition(snapshot, "sprint-1", domain.UserMemberID)
		if err != nil {
			t.Fatal(err)
		}

		if pos.SprintID != "sprint-1" {
			t.Errorf("expected SprintID=sprint-1, got %s", pos.SprintID)
		}
		if pos.Me.MemberID != domain.UserMemberID {
			t.Errorf("expected UserMemberID, got %s", pos.Me.MemberID)
		}
		if pos.Me.MemberName != "user" {
			t.Errorf("expected name=user, got %s", pos.Me.MemberName)
		}
		if len(pos.Workers) != len(snapshot.Members) {
			t.Errorf("user should see all members as workers: %+v", pos.Workers)
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

		_, err := BuildPosition(snapshot, "sprint-1", "nonexistent")
		if err == nil {
			t.Error("should return error for unknown member ID")
		}
	})
}

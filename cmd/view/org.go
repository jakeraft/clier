package view

type OrgInviteResult struct {
	Status string `json:"status"`
	Org    string `json:"org"`
	User   string `json:"user"`
	Role   string `json:"role"`
}

type OrgRemoveResult struct {
	Status  string `json:"status"`
	Org     string `json:"org"`
	Removed string `json:"removed"`
}

func OrgInviteOf(org, user, role string) OrgInviteResult {
	return OrgInviteResult{
		Status: "invited",
		Org:    org,
		User:   user,
		Role:   role,
	}
}

func OrgRemoveOf(org, removed string) OrgRemoveResult {
	return OrgRemoveResult{
		Status:  "removed",
		Org:     org,
		Removed: removed,
	}
}

package api

import "fmt"

func (c *Client) CreateOrg(body CreateOrgRequest) (*OrgResponse, error) {
	var r OrgResponse
	return &r, c.post("/api/v1/orgs", body, &r)
}

func (c *Client) DeleteOrg(owner string) error {
	return c.delete("/api/v1/orgs/" + owner)
}

func (c *Client) ListMyOrgs() ([]OrgResponse, error) {
	var r []OrgResponse
	return r, c.get("/api/v1/user/orgs", &r)
}

func (c *Client) ListOrgMembers(owner string) ([]OrgMemberResponse, error) {
	var r []OrgMemberResponse
	return r, c.get(fmt.Sprintf("/api/v1/orgs/%s/org-members", owner), &r)
}

func (c *Client) InviteOrgMember(owner string, body InviteMemberRequest) error {
	return c.post(fmt.Sprintf("/api/v1/orgs/%s/org-members", owner), body, nil)
}

func (c *Client) RemoveOrgMember(owner, name string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s/org-members/%s", owner, name))
}

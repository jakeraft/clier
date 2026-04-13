package api

import "fmt"

// --- Unified Read ---

func (c *Client) GetResource(owner, name string) (*ResourceResponse, error) {
	var r ResourceResponse
	return &r, c.get(fmt.Sprintf("/api/v1/orgs/%s/resources/%s", owner, name), &r)
}

func (c *Client) ListResources(owner string, opts ListOptions) (*ListResponse, error) {
	var r ListResponse
	path := fmt.Sprintf("/api/v1/orgs/%s/resources?%s", owner, buildListQuery(opts))
	return &r, c.get(path, &r)
}

func (c *Client) ListPublicResources(opts ListOptions) (*ListResponse, error) {
	var r ListResponse
	path := "/api/v1/resources?" + buildListQuery(opts)
	return &r, c.get(path, &r)
}

func (c *Client) ListResourceVersions(owner, name string) ([]ResourceVersionResponse, error) {
	var r []ResourceVersionResponse
	return r, c.get(fmt.Sprintf("/api/v1/orgs/%s/resources/%s/versions", owner, name), &r)
}

func (c *Client) GetResourceVersion(owner, name string, version int) (*ResourceVersionResponse, error) {
	var r ResourceVersionResponse
	return &r, c.get(fmt.Sprintf("/api/v1/orgs/%s/resources/%s/versions/%d", owner, name, version), &r)
}

// --- Generic Write ---

func (c *Client) CreateResource(kind ResourceKind, owner string, body any) (*ResourceResponse, error) {
	var r ResourceResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/%s", owner, kind.urlPath()), body, &r)
}

func (c *Client) UpdateResource(kind ResourceKind, owner, name string, body any) (*ResourceResponse, error) {
	var r ResourceResponse
	return &r, c.put(fmt.Sprintf("/api/v1/orgs/%s/%s/%s", owner, kind.urlPath(), name), body, &r)
}

func (c *Client) PatchResource(kind ResourceKind, owner, name string, body any) (*ResourceResponse, error) {
	var r ResourceResponse
	return &r, c.patch(fmt.Sprintf("/api/v1/orgs/%s/%s/%s", owner, kind.urlPath(), name), body, &r)
}

func (c *Client) DeleteResource(kind ResourceKind, owner, name string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s/%s/%s", owner, kind.urlPath(), name))
}

func (c *Client) ForkResource(kind ResourceKind, owner, name string) (*ResourceResponse, error) {
	var r ResourceResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/%s/%s/fork", owner, kind.urlPath(), name), nil, &r)
}

// --- Helpers ---

func buildListQuery(opts ListOptions) string {
	q := ""
	sep := ""
	if opts.Kind != "" {
		q += sep + "kind=" + opts.Kind
		sep = "&"
	}
	if opts.Query != "" {
		q += sep + "q=" + opts.Query
		sep = "&"
	}
	if opts.Limit > 0 {
		q += sep + fmt.Sprintf("limit=%d", opts.Limit)
		sep = "&"
	}
	if opts.Offset > 0 {
		q += sep + fmt.Sprintf("offset=%d", opts.Offset)
	}
	return q
}

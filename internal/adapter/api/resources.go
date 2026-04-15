package api

import (
	"fmt"
	"net/url"
	"strconv"
)

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

func (c *Client) CopyResource(kind ResourceKind, owner, name string) (*ResourceResponse, error) {
	var r ResourceResponse
	return &r, c.post(fmt.Sprintf("/api/v1/orgs/%s/%s/%s/copy", owner, kind.urlPath(), name), nil, &r)
}

// --- Stars ---

func (c *Client) StarResource(owner, name string) error {
	return c.do("PUT", fmt.Sprintf("/api/v1/orgs/%s/resources/%s/star", owner, name), nil, nil)
}

func (c *Client) UnstarResource(owner, name string) error {
	return c.delete(fmt.Sprintf("/api/v1/orgs/%s/resources/%s/star", owner, name))
}

// --- Helpers ---

func buildListQuery(opts ListOptions) string {
	v := url.Values{}
	if opts.Kind != "" {
		v.Set("kind", opts.Kind)
	}
	if opts.Query != "" {
		v.Set("q", opts.Query)
	}
	if opts.Uses != "" {
		v.Set("uses", opts.Uses)
	}
	if opts.Starred != nil {
		v.Set("starred", strconv.FormatBool(*opts.Starred))
	}
	if opts.Limit > 0 {
		v.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		v.Set("offset", strconv.Itoa(opts.Offset))
	}
	if opts.Sort != "" {
		v.Set("sort", opts.Sort)
	}
	if opts.Order != "" {
		v.Set("order", opts.Order)
	}
	return v.Encode()
}

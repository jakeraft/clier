package catalog

import remoteapi "github.com/jakeraft/clier/internal/adapter/api"

type RemoteCatalogClient interface {
	GetResource(owner, name string) (*remoteapi.ResourceResponse, error)
	GetResourceVersion(owner, name string, version int) (*remoteapi.ResourceVersionResponse, error)
	ListResources(owner string, opts remoteapi.ListOptions) (*remoteapi.ListResponse, error)
	ListPublicResources(opts remoteapi.ListOptions) (*remoteapi.ListResponse, error)
	ListResourceVersions(owner, name string) ([]remoteapi.ResourceVersionResponse, error)
	CreateResource(kind remoteapi.ResourceKind, owner string, body any) (*remoteapi.ResourceResponse, error)
	PatchResource(kind remoteapi.ResourceKind, owner, name string, body any) (*remoteapi.ResourceResponse, error)
	DeleteResource(kind remoteapi.ResourceKind, owner, name string) error
	ForkResource(kind remoteapi.ResourceKind, owner, name string) (*remoteapi.ResourceResponse, error)
	CreateOrg(body remoteapi.CreateOrgRequest) (*remoteapi.OrgResponse, error)
	DeleteOrg(owner string) error
	ListMyOrgs() ([]remoteapi.OrgResponse, error)
	ListOrgMembers(owner string) ([]remoteapi.OrgMemberResponse, error)
	InviteOrgMember(owner string, body remoteapi.InviteMemberRequest) error
	RemoveOrgMember(owner, name string) error
}

type Service struct {
	client RemoteCatalogClient
}

func New(client RemoteCatalogClient) *Service {
	return &Service{client: client}
}

func (s *Service) GetResource(owner, name string) (*remoteapi.ResourceResponse, error) {
	return s.client.GetResource(owner, name)
}

func (s *Service) GetResourceVersion(owner, name string, version int) (*remoteapi.ResourceVersionResponse, error) {
	return s.client.GetResourceVersion(owner, name, version)
}

func (s *Service) ListResources(owner string, opts remoteapi.ListOptions) (*remoteapi.ListResponse, error) {
	return s.client.ListResources(owner, opts)
}

func (s *Service) ListPublicResources(opts remoteapi.ListOptions) (*remoteapi.ListResponse, error) {
	return s.client.ListPublicResources(opts)
}

func (s *Service) ListResourceVersions(owner, name string) ([]remoteapi.ResourceVersionResponse, error) {
	return s.client.ListResourceVersions(owner, name)
}

func (s *Service) CreateResource(kind remoteapi.ResourceKind, owner string, body any) (*remoteapi.ResourceResponse, error) {
	return s.client.CreateResource(kind, owner, body)
}

func (s *Service) PatchResource(kind remoteapi.ResourceKind, owner, name string, body any) (*remoteapi.ResourceResponse, error) {
	return s.client.PatchResource(kind, owner, name, body)
}

func (s *Service) DeleteResource(kind remoteapi.ResourceKind, owner, name string) error {
	return s.client.DeleteResource(kind, owner, name)
}

func (s *Service) ForkResource(kind remoteapi.ResourceKind, owner, name string) (*remoteapi.ResourceResponse, error) {
	return s.client.ForkResource(kind, owner, name)
}

func (s *Service) CreateOrg(body remoteapi.CreateOrgRequest) (*remoteapi.OrgResponse, error) {
	return s.client.CreateOrg(body)
}

func (s *Service) DeleteOrg(owner string) error {
	return s.client.DeleteOrg(owner)
}

func (s *Service) ListMyOrgs() ([]remoteapi.OrgResponse, error) {
	return s.client.ListMyOrgs()
}

func (s *Service) ListOrgMembers(owner string) ([]remoteapi.OrgMemberResponse, error) {
	return s.client.ListOrgMembers(owner)
}

func (s *Service) InviteOrgMember(owner string, body remoteapi.InviteMemberRequest) error {
	return s.client.InviteOrgMember(owner, body)
}

func (s *Service) RemoveOrgMember(owner, name string) error {
	return s.client.RemoveOrgMember(owner, name)
}

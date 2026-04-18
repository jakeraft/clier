package auth

import (
	"fmt"
	"time"

	remoteapi "github.com/jakeraft/clier/internal/adapter/api"
	"github.com/jakeraft/clier/internal/domain"
)

type RemoteAuthClient interface {
	RequestDeviceCode() (*remoteapi.DeviceCodeResponse, error)
	PollDeviceAuth(deviceCode string) (*remoteapi.DevicePollResponse, error)
	GetCurrentUser() (*remoteapi.UserResponse, error)
	Logout() error
}

type Service struct {
	client RemoteAuthClient
	sleep  func(time.Duration)
	now    func() time.Time
}

type LoginPrompt struct {
	UserCode        string
	VerificationURI string
}

func NewService(client RemoteAuthClient) *Service {
	return &Service{
		client: client,
		sleep:  time.Sleep,
		now:    time.Now,
	}
}

func (s *Service) Login(credentialsPath string, notify func(LoginPrompt)) (*remoteapi.UserResponse, error) {
	resp, err := s.client.RequestDeviceCode()
	if err != nil {
		return nil, fmt.Errorf("failed to start login: %w", err)
	}
	if notify != nil {
		notify(LoginPrompt{
			UserCode:        resp.UserCode,
			VerificationURI: resp.VerificationURI,
		})
	}

	interval := time.Duration(resp.Interval) * time.Second
	if interval == 0 {
		interval = 5 * time.Second
	}
	deadline := s.now().Add(time.Duration(resp.ExpiresIn) * time.Second)

	for s.now().Before(deadline) {
		s.sleep(interval)

		poll, err := s.client.PollDeviceAuth(resp.DeviceCode)
		if err != nil {
			return nil, fmt.Errorf("poll failed: %w", err)
		}

		if poll.AccessToken != "" && poll.User != nil {
			creds := &Credentials{
				Token: poll.AccessToken,
				Login: poll.User.Name,
			}
			if err := Save(credentialsPath, creds); err != nil {
				return nil, fmt.Errorf("failed to save credentials: %w", err)
			}
			return poll.User, nil
		}

		if poll.Status == "slow_down" {
			interval += 5 * time.Second
		}
	}

	return nil, &domain.Fault{Kind: domain.KindAuthTimeout}
}

func (s *Service) Logout(credentialsPath string) error {
	if err := s.client.Logout(); err != nil {
		return err
	}
	return Delete(credentialsPath)
}

func (s *Service) Status(credentialsPath string) (*remoteapi.UserResponse, error) {
	creds, err := Load(credentialsPath)
	if err != nil {
		return nil, err
	}
	if creds == nil {
		return nil, &domain.Fault{Kind: domain.KindAuthRequired}
	}
	return s.client.GetCurrentUser()
}

func (s *Service) Token(credentialsPath string) (string, error) {
	creds, err := Load(credentialsPath)
	if err != nil {
		return "", err
	}
	return creds.Token, nil
}

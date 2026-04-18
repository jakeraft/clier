package auth

import (
	"strings"
	"testing"
	"time"

	remoteapi "github.com/jakeraft/clier/internal/adapter/api"
)

type fakeRemoteAuthClient struct {
	requestDeviceCodeFn func() (*remoteapi.DeviceCodeResponse, error)
	pollDeviceAuthFn    func(deviceCode string) (*remoteapi.DevicePollResponse, error)
	getCurrentUserFn    func() (*remoteapi.UserResponse, error)
	logoutFn            func() error
}

func (f fakeRemoteAuthClient) RequestDeviceCode() (*remoteapi.DeviceCodeResponse, error) {
	return f.requestDeviceCodeFn()
}

func (f fakeRemoteAuthClient) PollDeviceAuth(deviceCode string) (*remoteapi.DevicePollResponse, error) {
	return f.pollDeviceAuthFn(deviceCode)
}

func (f fakeRemoteAuthClient) GetCurrentUser() (*remoteapi.UserResponse, error) {
	if f.getCurrentUserFn == nil {
		return nil, nil
	}
	return f.getCurrentUserFn()
}

func (f fakeRemoteAuthClient) Logout() error {
	if f.logoutFn == nil {
		return nil
	}
	return f.logoutFn()
}

func TestLoginFailsImmediatelyOnPollError(t *testing.T) {
	t.Parallel()

	svc := NewService(fakeRemoteAuthClient{
		requestDeviceCodeFn: func() (*remoteapi.DeviceCodeResponse, error) {
			return &remoteapi.DeviceCodeResponse{
				DeviceCode:      "device-code",
				UserCode:        "user-code",
				VerificationURI: "https://example.com/device",
				ExpiresIn:       30,
				Interval:        1,
			}, nil
		},
		pollDeviceAuthFn: func(deviceCode string) (*remoteapi.DevicePollResponse, error) {
			return nil, &remoteapi.Error{
				StatusCode: 400,
				Status: &remoteapi.Status{
					Reason:  remoteapi.ReasonAuthFailed,
					Message: "device flow denied",
				},
			}
		},
	})
	svc.sleep = func(time.Duration) {}
	nowCalls := 0
	start := time.Unix(0, 0)
	svc.now = func() time.Time {
		tm := start.Add(time.Duration(nowCalls) * time.Second)
		nowCalls++
		return tm
	}

	_, err := svc.Login(t.TempDir()+"/credentials.json", nil)
	if err == nil {
		t.Fatal("expected poll failure")
	}
	if !strings.Contains(err.Error(), "poll failed") {
		t.Fatalf("got %v, want poll failure", err)
	}
}

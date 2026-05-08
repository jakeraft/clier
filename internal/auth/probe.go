package auth

import (
	"errors"

	"github.com/jakeraft/clier/internal/api"
)

// CodeUnauthenticated 는 server 의 RFC 9457 Problem `code` 슬러그 — auth
// flow 가 잡아 status 401 로 실어 보내는 카테고리. CLI 가 raw status code
// 매직 (== 401) 대신 이 슬러그로 의미를 분기해야 server 가 미래에 다른
// 401 카테고리를 도입했을 때 silently 같은 분기에 들어가지 않는다.
const CodeUnauthenticated = "UNAUTHENTICATED"

// SessionProbe 는 ProbeSession 의 결과 — auth status 출력 모양과 1:1.
// Reason 의 빈 문자열은 "추가 신호 없음" sentinel.
type SessionProbe struct {
	LoggedIn bool
	Login    string
	Reason   string
}

// ProbeSession 은 persistedLogin 의 세션이 server 에서 여전히 유효한지
// 묻는다. 서버 envelope 의 `code` 슬러그를 분기 기준으로 써서 raw status
// code 매직을 피한다.
//
// - 200: (true, ns.Name, "")
// - 401 + code=UNAUTHENTICATED: (false, persistedLogin, "session_expired")
// - 그 외 오류: 그대로 caller 에 surface (network 실패 / 서버 다운 등 —
//   사용자가 무엇을 해야 할지가 다른 문제)
//
// status / login fast-path 양쪽이 같은 helper 를 호출해 raw status code
// 매직과 client 인스턴스 재생성이 한 자리로 모이게 한다.
func ProbeSession(client *api.Client, persistedLogin string) (SessionProbe, error) {
	ns, err := client.AuthMe()
	if err == nil {
		return SessionProbe{LoggedIn: true, Login: ns.Name}, nil
	}
	var apiErr *api.Error
	if errors.As(err, &apiErr) && apiErr.Code() == CodeUnauthenticated {
		return SessionProbe{LoggedIn: false, Login: persistedLogin, Reason: "session_expired"}, nil
	}
	return SessionProbe{}, err
}

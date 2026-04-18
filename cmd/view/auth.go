package view

type AuthLoginResult struct {
	Status string `json:"status"`
	Login  string `json:"login"`
}

type AuthLogoutResult struct {
	Status string `json:"status"`
}

type AuthStatusResult struct {
	Login string `json:"login"`
}

type AuthTokenResult struct {
	Token string `json:"token"`
}

func AuthLoginOf(login string) AuthLoginResult {
	return AuthLoginResult{Status: "logged_in", Login: login}
}

func AuthLogoutOf() AuthLogoutResult {
	return AuthLogoutResult{Status: "logged_out"}
}

func AuthStatusOf(login string) AuthStatusResult {
	return AuthStatusResult{Login: login}
}

func AuthTokenOf(token string) AuthTokenResult {
	return AuthTokenResult{Token: token}
}

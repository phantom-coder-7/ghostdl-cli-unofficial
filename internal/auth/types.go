package auth

const (
	KindToken = "token"
)

type LoginData struct {
	Kind  string `json:"kind"`
	Token string `json:"token"`
}

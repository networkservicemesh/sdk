package defaultpolicies

import (
	"github.com/dgrijalva/jwt-go"
	"time"
)

const (
	KEY = "test"
)

func generateTokenWithExpireTime(expireTime time.Time) (string, error) {
	claims := jwt.StandardClaims{
		ExpiresAt: expireTime.Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(KEY))
}



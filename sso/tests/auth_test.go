package tests

import (
	"encoding/hex"
	as_jwt "sso/internal/lib/jwt"
	"sso/tests/suite"
	"testing"
	"time"

	ssov1 "github.com/CheckEZ/protos/gen/go/sso"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	deltaSeconds   = 1
	passDefaultLen = 10
	publicKeyPath  = "../tools/certs/jwt_public.pem"
)

func randomFakePassword() string {
	return gofakeit.Password(true, true, true, true, false, passDefaultLen)
}

func randomWallet() string {
	bytes := make([]byte, 20)

	for i := range bytes {
		bytes[i] = byte(gofakeit.Uint8())
	}

	return "0x" + hex.EncodeToString(bytes)
}

func TestRegisterLoginRefreshLogout_HappyPath(t *testing.T) {
	ctx, _suite := suite.New(t)

	email := gofakeit.Email()
	wallet := randomWallet()
	password1 := randomFakePassword()
	password2 := password1

	respReg, err := _suite.AuthClient.Register(ctx, &ssov1.RegisterRequest{
		Email:     email,
		Wallet:    wallet,
		Password1: password1,
		Password2: password2,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, respReg.GetUserId())

	respLogin, err := _suite.AuthClient.Login(ctx, &ssov1.LoginRequest{
		Email:    email,
		Password: password1,
	})
	require.NoError(t, err)

	accessToken := respLogin.GetAccessToken()
	refreshToken := respLogin.GetRefreshToken()

	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshToken)

	parseClaims := func(tokenString string) jwt.MapClaims {
		claims, err := as_jwt.DecodeJWT(tokenString, publicKeyPath)
		require.NoError(t, err)

		return claims
	}

	accessClaims := parseClaims(accessToken)
	refreshClaims := parseClaims(refreshToken)

	assert.Equal(t, respReg.GetUserId(), int64(accessClaims["uid"].(float64)))
	assert.Equal(t, email, accessClaims["email"].(string))
	assert.Equal(t, wallet, accessClaims["wallet"].(string))

	assert.Equal(t, respReg.GetUserId(), int64(refreshClaims["uid"].(float64)))
	assert.Equal(t, email, refreshClaims["email"].(string))
	assert.Equal(t, wallet, refreshClaims["wallet"].(string))

	loginTime := time.Now()
	assert.InDelta(t, loginTime.Add(_suite.Cfg.AccessTTL).Unix(), accessClaims["exp"].(float64), deltaSeconds)
	assert.InDelta(t, loginTime.Add(_suite.Cfg.RefreshTTL).Unix(), refreshClaims["exp"].(float64), deltaSeconds)

	respRefresh, err := _suite.AuthClient.Refresh(ctx, &ssov1.RefreshRequest{
		RefreshToken: refreshToken,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, respRefresh.GetAccessToken())

	_, err = _suite.AuthClient.Logout(ctx, &ssov1.LogoutRequest{
		RefreshToken: refreshToken,
	})
	require.NoError(t, err)
}

func TestRegister_DuplicatedRegistration(t *testing.T) {
	ctx, _suite := suite.New(t)

	email := gofakeit.Email()
	wallet := randomWallet()
	password1 := randomFakePassword()
	password2 := password1

	respReg, err := _suite.AuthClient.Register(ctx, &ssov1.RegisterRequest{
		Email:     email,
		Wallet:    wallet,
		Password1: password1,
		Password2: password2,
	})
	require.NoError(t, err)
	require.NotEmpty(t, respReg.GetUserId())

	respReg, err = _suite.AuthClient.Register(ctx, &ssov1.RegisterRequest{
		Email:     email,
		Wallet:    wallet,
		Password1: password1,
		Password2: password2,
	})
	require.Error(t, err)
	assert.Empty(t, respReg.GetUserId())
	assert.ErrorContains(t, err, "user with email already exists")
}

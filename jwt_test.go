package main

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTRoundTrip(t *testing.T) {
	token, err := GenerateJWT("student", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ValidateJWT(token, "test-secret")
	if err != nil || claims["user"] != "student" {
		t.Fatalf("round trip: claims=%v, error=%v", claims, err)
	}
	if _, err := ValidateJWT(token, "wrong-secret"); err == nil {
		t.Fatal("accepted a token with the wrong signature")
	}
}

func TestJWTRejectsInvalidClaimsAndAlgorithm(t *testing.T) {
	for _, test := range []struct {
		name   string
		method jwt.SigningMethod
		claims jwt.MapClaims
	}{
		{"expired", jwt.SigningMethodHS256, jwt.MapClaims{"user": "student", "exp": time.Now().Add(-time.Minute).Unix()}},
		{"missing expiration", jwt.SigningMethodHS256, jwt.MapClaims{"user": "student"}},
		{"invalid user", jwt.SigningMethodHS256, jwt.MapClaims{"user": 123, "exp": time.Now().Add(time.Minute).Unix()}},
		{"missing user", jwt.SigningMethodHS256, jwt.MapClaims{"exp": time.Now().Add(time.Minute).Unix()}},
		{"wrong algorithm", jwt.SigningMethodHS512, jwt.MapClaims{"user": "student", "exp": time.Now().Add(time.Minute).Unix()}},
	} {
		t.Run(test.name, func(t *testing.T) {
			token, err := jwt.NewWithClaims(test.method, test.claims).SignedString([]byte("test-secret"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ValidateJWT(token, "test-secret"); err == nil {
				t.Fatal("accepted invalid token")
			}
		})
	}
}

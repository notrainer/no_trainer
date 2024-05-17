package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
)

func GenerateJWT(username string, secret string) (string, error) {
	//make sure you use HS256 for signingMethods
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		//exp means expiration time. set it 1 minutes to check if it works.
		"exp": time.Now().Add(1 * time.Minute).Unix(),
		//set authorized true
		"authorized": true,
		//as a user we set username from the input you may want to set something else..
		"user": username,
	})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return tokenString, nil
}

func ValidateJWT(tokenString string, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

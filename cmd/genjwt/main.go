package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	secret := "your-256-bit-secret-key-here-change-in-production"
	if len(os.Args) > 1 {
		secret = os.Args[1]
	}

	npm := "2211700006"
	if len(os.Args) > 2 {
		npm = os.Args[2]
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"npm": npm,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(tokenString)
}

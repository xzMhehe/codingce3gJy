package main

import (
	"fmt"

	"qqjiayuan/server/internal/config"
	"qqjiayuan/server/pkg/authutil"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		panic(err)
	}
	t, err := authutil.GenerateToken(10000, "站长小Q", cfg.Jwt.Secret, 24)
	if err != nil {
		panic(err)
	}
	fmt.Println(t)
}

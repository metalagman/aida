package cmd

import "github.com/joho/godotenv"

func initDotEnv() {
	_ = godotenv.Load()
}

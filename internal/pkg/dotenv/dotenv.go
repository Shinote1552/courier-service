package dotenv

import (
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func Load() error {
	err := godotenv.Load()
	if err != nil {
		return err
	}

	var portFlag string
	flag.StringVar(&portFlag, "port", "", "Server port (overrides PORT environment variable)")
	flag.Parse()

	if portFlag != "" {
		err := os.Setenv("PORT", portFlag)
		if err != nil {
			return fmt.Errorf("failed to set PORT environment variable: %w", err)
		}
	}
	return nil
}

package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/arfajhf/copytygo/v4/auth"
	"github.com/arfajhf/copytygo/v4/config"
	"github.com/arfajhf/copytygo/v4/database"
)

func AuthAdmin(args []string) error {
	if len(args) != 1 || !strings.Contains(args[0], "@") {
		return fmt.Errorf("usage: ctg auth:admin <registered-email>")
	}
	if err := config.LoadEnv(".env"); err != nil {
		return err
	}
	defer database.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := auth.PromoteAdmin(ctx, args[0]); err != nil {
		return err
	}
	fmt.Printf("Administrator role granted to %s. Open /dashboard to manage users.\n", args[0])
	return nil
}

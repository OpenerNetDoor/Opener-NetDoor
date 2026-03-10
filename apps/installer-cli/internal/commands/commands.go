package commands

import "fmt"

func Run(command string) error {
	switch command {
	case "install", "upgrade", "rollback", "uninstall", "backup", "restore", "healthcheck", "reset-password", "add-node", "rotate-keys", "migrate":
		fmt.Printf("TODO: execute %s workflow\n", command)
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

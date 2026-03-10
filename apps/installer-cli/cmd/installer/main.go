package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/opener-netdoor/opener-netdoor/apps/installer-cli/internal/commands"
)

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Println("usage: installer <install|upgrade|rollback|uninstall|backup|restore|healthcheck|reset-password|add-node|rotate-keys|migrate>")
		os.Exit(2)
	}
	cmd := flag.Arg(0)
	if err := commands.Run(cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}



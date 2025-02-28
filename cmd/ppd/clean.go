package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/stratosnet/sds/pp/setting"
)

// time: Feb 3 2025 16:56 UTC
const DEFAULT_UNIX_TIME = 1738601804

func cleanStorage(cmd *cobra.Command, _ []string) error {
	var s string

	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Please confirm to clean the old data in storage: [Y/n]")
		s, _ = r.ReadString('\n')
		ss := strings.Split(s, "\n")
		if ss[0] == "N" || ss[0] == "n" {
			fmt.Println("Data in storage is NOT cleaned.")
			os.Exit(0)
		}
		if ss[0] == "Y" || ss[0] == "y" || ss[0] == "" {
			break
		}
	}

	cmdString := fmt.Sprintf(`find %s -type f -not -newermt "$(date -d @%d)" -exec sudo rm -rf {} \;`, setting.Config.Home.StoragePath, uint64(DEFAULT_UNIX_TIME))
	c := exec.Command("sh", "-c", cmdString)
	out, err := c.CombinedOutput()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	// Print output (if any)
	fmt.Println(string(out))
	return nil
}

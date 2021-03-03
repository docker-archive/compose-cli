/*
   Copyright 2020 Docker Compose CLI authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/docker/compose-cli/api/context/store"
	"github.com/docker/compose-cli/cli/cmd/compose"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please inform one of the supported shells to generate the completion. [bash,zsh,fish,powershell]")
		os.Exit(1)
	}
	root := &cobra.Command{
		Use: "docker",
	}
	c := compose.Command(store.DefaultContextType)
	root.AddCommand(c)
	var err error
	switch os.Args[1] {
	case "bash":
		err = root.GenBashCompletion(os.Stdout)
	case "zsh":
		err = root.GenZshCompletion(os.Stdout)
	case "fish":
		err = root.GenFishCompletion(os.Stdout, true)
	case "powershell":
		err = root.GenPowerShellCompletion(os.Stdout)
	default:
		fmt.Printf("Unsupported shell: %q", os.Args[1])
		os.Exit(1)
	}
	if err != nil {
		panic(err)
	}
}

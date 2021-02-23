/*
   Copyright 2021 Docker Compose CLI authors

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

package etchosts

import (
	"fmt"
	"os"
	"strings"
)

// SetHostNames appends hosts aliases for loopback address to etc/host file
func SetHostNames(file string, hosts ...string) error {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	fmt.Println("Setting local hosts for " + strings.Join(hosts, ", "))
	for _, host := range hosts {
		_, err = f.WriteString("\n127.0.0.1 " + host)
		if err != nil {
			return err
		}
	}
	_, err = f.WriteString("\n")
	return err
}

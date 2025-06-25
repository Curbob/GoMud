package flags

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func HandleFlags(serverVersion string) {

	var portsearch string
	var showVersion bool

	flag.StringVar(&portsearch, "port-search", "", "Search for the first 10 open ports: -port-search=30000-40000")
	flag.BoolVar(&showVersion, "version", false, "Display the current binary version")

	flag.Parse()

	if showVersion {
		fmt.Println(serverVersion)
		os.Exit(0)
	}

	if portsearch != `` {
		doPortSearch(portsearch)
		os.Exit(0)
	}
}

func doPortSearch(portRangeStr string) {
	portRange := strings.Split(portRangeStr, `-`)

	if len(portRange) < 2 {
		mudlog.Error("-port-search", "error", "Invalid port range specified")
		return
	}

	portRangeStart, _ := strconv.Atoi(portRange[0])
	portRangeEnd, _ := strconv.Atoi(portRange[1])

	if portRangeStart == 0 || portRangeEnd == 0 || portRangeStart >= portRangeEnd {
		mudlog.Error("-port-search", "error", "Invalid port range specified")
		return
	}

	mudlog.Info("-port-search", "message", fmt.Sprintf("Searching for first 10 available ports between %d and %d", portRangeStart, portRangeEnd))

	foundPorts := 0
	for i := portRangeStart; i < portRangeEnd; i++ {

		if !isPortInUse(i) {
			mudlog.Info("-port-search", "message", "Found port", "port", i)
			foundPorts++
		}
		if foundPorts >= 10 {
			break
		}
	}

	mudlog.Info("-port-search", "message", fmt.Sprintf("Found %d available ports", foundPorts))

}

func isPortInUse(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return true
	}
	ln.Close()
	return false
}

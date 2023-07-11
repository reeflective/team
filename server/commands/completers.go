package commands

import (
	"fmt"
	"net"
	"strings"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/server"
	"github.com/rsteube/carapace"
)

// interfacesCompleter completes interface addresses on the client host.
func interfacesCompleter() carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		ifaces, err := net.Interfaces()
		if err != nil {
			return carapace.ActionMessage("failed to get net interfaces: %s", err.Error())
		}

		results := make([]string, 0)

		for _, i := range ifaces {
			addrs, err := i.Addrs()
			if err != nil {
				continue
			}

			for _, a := range addrs {
				switch v := a.(type) {
				case *net.IPAddr:
					results = append(results, v.IP.String())
				case *net.IPNet:
					results = append(results, v.IP.String())
				default:
					results = append(results, v.String())
				}
			}
		}

		return carapace.ActionValues(results...).Tag("client interfaces").NoSpace(':')
	})
}

// userCompleter completes usernames of the application teamserver.
func userCompleter(client *client.Client, server *server.Server) carapace.CompletionCallback {
	return func(c carapace.Context) carapace.Action {
		users, err := client.Users()
		if err != nil {
			return carapace.ActionMessage("Failed to get users: %s", err)
		}

		results := make([]string, len(users))
		for _, user := range users {
			results = append(results, strings.TrimSpace(user.Name))
		}

		if len(results) == 0 {
			return carapace.ActionMessage(fmt.Sprintf("%s teamserver has no users", server.Name()))
		}

		return carapace.ActionValues(results...).Tag(fmt.Sprintf("%s teamserver users", server.Name()))
	}
}

// listenerIDCompleter completes ID for running teamserver listeners.
func listenerIDCompleter(client *client.Client, server *server.Server) carapace.CompletionCallback {
	return func(c carapace.Context) carapace.Action {
		listeners := server.Listeners()

		var results []string
		for _, ln := range listeners {
			results = append(results, strings.TrimSpace(formatSmallID(ln.ID)))
			results = append(results, fmt.Sprintf("[%s] (%s)", ln.Description, "Up"))
		}

		if len(results) == 0 {
			return carapace.ActionMessage(fmt.Sprintf("no listeners running for %s teamserver", server.Name()))
		}

		return carapace.ActionValuesDescribed(results...).Tag(fmt.Sprintf("%s teamserver listeners", server.Name()))
	}
}

// listenerTypeCompleter completes the different types of teamserver listener/handler stacks available.
func listenerTypeCompleter(client *client.Client, server *server.Server) carapace.CompletionCallback {
	return func(c carapace.Context) carapace.Action {
		listeners := server.Handlers()

		var results []string
		for _, ln := range listeners {
			results = append(results, strings.TrimSpace(ln.Name()))
		}

		if len(results) == 0 {
			return carapace.ActionMessage(fmt.Sprintf("no additional listener types for %s teamserver", server.Name()))
		}

		return carapace.ActionValues(results...).Tag(fmt.Sprintf("%s teamserver listener types", server.Name()))
	}
}

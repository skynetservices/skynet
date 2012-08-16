package main

import (
	"fmt"
	"github.com/bketelsen/skynet"
	"github.com/bketelsen/skynet/client"
	"github.com/kballard/go-shellquote"
	"os"
	"text/template"
)

// Remote() uses the SkynetDaemon service to remotely manage services.
func Remote(q *client.Query, args []string) {
	if len(args) == 0 {
		remoteHelp()
		return
	}
	switch args[0] {
	case "list":
		remoteList(q)
	case "deploy":
		if len(args) < 2 {
			fmt.Printf("Must specify a service path")
			remoteHelp()
			return
		}
		servicePath := args[1]
		serviceArgs := args[2:]
		remoteDeploy(q, servicePath, serviceArgs)
	case "start":
		if len(args) != 2 {
			fmt.Printf("Must specify a service UUID")
			remoteHelp()
			return
		}
		uuid := args[1]
		remoteStart(q, uuid)
	case "startall":
		if len(args) != 1 {
			remoteHelp()
			return
		}
		remoteStartAll(q)
	case "stop":
		if len(args) != 2 {
			fmt.Printf("Must specify a service UUID")
			remoteHelp()
			return
		}
		uuid := args[1]
		remoteStop(q, uuid)
	case "stopall":
		if len(args) != 1 {
			remoteHelp()
			return
		}
		remoteStopAll(q)
	case "restart":
		if len(args) != 2 {
			fmt.Printf("Must specify a service UUID")
			remoteHelp()
			return
		}
		uuid := args[1]
		remoteRestart(q, uuid)
	case "restartall":
		if len(args) != 1 {
			remoteHelp()
			return
		}
		remoteRestartAll(q)
	case "help":
		remoteHelp()
	default:
		fmt.Printf("Unknown command %q", args[0])
		remoteHelp()
	}
	return
}

func getDaemonServiceClient(q *client.Query) (c *client.Client, service *client.ServiceClient) {
	config, _ := skynet.GetClientConfigFromFlags(os.Args...)

	config.Log = skynet.NewConsoleLogger(os.Stderr)

	c = client.NewClient(config)

	registered := true
	query := &client.Query{
		DoozerConn: c.DoozerConn,
		Service:    "SkynetDaemon",
		//Host:       "127.0.0.1",
		Registered: &registered,
	}
	service = c.GetServiceFromQuery(query)
	return
}

var listTemplate = template.Must(template.New("").Parse(
	`{{range .Services}}UUID: {{.UUID}}
 {{.ServicePath}}
 {{.Args}}
 {{if .Running}}RUNNING{{else}}HALTED{{end}}
{{end}}
`))

func remoteList(q *client.Query) {
	_, service := getDaemonServiceClient(q)

	// This on the other hand will fail if it can't find a service to connect to
	var response ListSubServicesOut
	err := service.Send(nil, "ListSubServices", ListSubServicesIn{}, &response)

	if err != nil {
		fmt.Println(err)
		return
	}

	listTemplate.Execute(os.Stdout, response)
}

var deployTemplate = template.Must(template.New("").Parse(
	`Deployed service with UUID {{.UUID}}.
`))

func remoteDeploy(q *client.Query, servicePath string, serviceArgs []string) {
	_, service := getDaemonServiceClient(q)

	in := DeployIn{
		ServicePath: servicePath,
		Args:        shellquote.Join(serviceArgs...),
	}
	var out DeployOut

	err := service.Send(nil, "Deploy", in, &out)

	if err != nil {
		fmt.Println(err)
		return
	}

	deployTemplate.Execute(os.Stdout, out)
}

var startTemplate = template.Must(template.New("").Parse(
	`{{if .Ok}}Started service with UUID {{.UUID}}.
{{else}}Service with UUID {{.UUID}} is already running.
{{end}}`))

func remoteStart(q *client.Query, uuid string) {
	_, service := getDaemonServiceClient(q)

	// This on the other hand will fail if it can't find a service to connect to
	var in = StartSubServiceIn{UUID: uuid}
	var out StartSubServiceOut
	err := service.Send(nil, "StartSubService", in, &out)

	if err != nil {
		fmt.Println(err)
		return
	}

	startTemplate.Execute(os.Stdout, out)
}

var startallTemplate = template.Must(template.New("").Parse(
	`Started {{.Count}} services.
{{range .Starts}} {{.UUID}}: {{if .Ok}}STARTED{{else}}ALREADY STARTED{{end}}
{{end}}`))

func remoteStartAll(q *client.Query) {
	_, service := getDaemonServiceClient(q)
	var in StartAllSubServicesIn
	var out StartAllSubServicesOut
	err := service.Send(nil, "StartAllSubServices", in, &out)

	if err != nil {
		fmt.Println(err)
		return
	}

	startallTemplate.Execute(os.Stdout, out)
}

var stopTemplate = template.Must(template.New("").Parse(
	`{{if .Ok}}Stopped service with UUID {{.UUID}}.
{{else}}Service with UUID {{.UUID}} is already stopped.
{{end}}`))

func remoteStop(q *client.Query, uuid string) {
	_, service := getDaemonServiceClient(q)

	// This on the other hand will fail if it can't find a service to connect to
	var in = StopSubServiceIn{UUID: uuid}
	var out StopSubServiceOut
	err := service.Send(nil, "StopSubService", in, &out)

	if err != nil {
		fmt.Println(err)
		return
	}

	stopTemplate.Execute(os.Stdout, out)
}

var stopallTemplate = template.Must(template.New("").Parse(
	`Stopped {{.Count}} services.
{{range .Stops}} {{.UUID}}: {{if .Ok}}STOPPED{{else}}ALREADY STOPPED{{end}}
{{end}}`))

func remoteStopAll(q *client.Query) {
	_, service := getDaemonServiceClient(q)
	var in StopAllSubServicesIn
	var out StopAllSubServicesOut
	err := service.Send(nil, "StopAllSubServices", in, &out)

	if err != nil {
		fmt.Println(err)
		return
	}

	stopallTemplate.Execute(os.Stdout, out)
}

var restartTemplate = template.Must(template.New("").Parse(
	`Restarted service with UUID {{.UUID}}.
`))

func remoteRestart(q *client.Query, uuid string) {
	_, service := getDaemonServiceClient(q)

	// This on the other hand will fail if it can't find a service to connect to
	var in = RestartSubServiceIn{UUID: uuid}
	var out RestartSubServiceOut
	err := service.Send(nil, "RestartSubService", in, &out)

	if err != nil {
		fmt.Println(err)
		return
	}

	restartTemplate.Execute(os.Stdout, out)
}

var restartallTemplate = template.Must(template.New("").Parse(
	`Restarted {{len .Restarts}} services.
{{range .Restarts}} {{.UUID}}
{{end}}`))

func remoteRestartAll(q *client.Query) {
	_, service := getDaemonServiceClient(q)
	var in RestartAllSubServicesIn
	var out RestartAllSubServicesOut
	err := service.Send(nil, "RestartAllSubServices", in, &out)

	if err != nil {
		fmt.Println(err)
		return
	}

	restartallTemplate.Execute(os.Stdout, out)
}

func remoteHelp() {
	fmt.Println(`remote commands:
	help
		- Print this help text.
	list
		- List all services currently being run by this daemon, with their uuids.
	deploy [service path] [arguments]
		- Deploy the service specified by the path, launched with the given arguments.
		  The uuid of the service will be printed.
	start [uuid]
		- Start the service assined to the given uuid.
	startall
		- Start all services.
	stop [uuid]
		- Stop the service assined to the given uuid.
	stopall
		- Stop all services.
	restart [uuid]
		- Restart the service assined to the given uuid.
	restartall
		- Restart all services.
`)
}

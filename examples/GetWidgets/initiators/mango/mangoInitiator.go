package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/paulbellamy/mango"
	"github.com/bketelsen/skynet/skylib"
	"log"
	"myStartup"
	"os"
	"rpc"
	"template"
)

const sName = "Initiator.Web"

const homeTemplate = `<!DOCTYPE html PUBLIC '-//W3C//DTD XHTML 1.0 Transitional//EN' 'http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd'><html xmlns='http://www.w3.org/1999/xhtml' xml:lang='en' lang='en'><head></head><body id='body'><form action='/new' method='POST'><div>Your Input Value<input type='text' name='YourInputValue' value=''></input></div>	</form></body></html>`
const responseTemplate = `<!DOCTYPE html PUBLIC '-//W3C//DTD XHTML 1.0 Transitional//EN' 'http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd'><html xmlns='http://www.w3.org/1999/xhtml' xml:lang='en' lang='en'><head></head><body id='body'>{.repeated section resp.Errors} There were errors:<br/>{@}<br/>{.end}<div>Your Output Value: {resp.YourOutputValue}</div>	</body></html>	`

// Call the RPC service on the router to process the GetUserDataRequest.
func submitGetUserDataRequest(cr *myStartup.GetUserDataRequest) (*myStartup.GetUserDataResponse, os.Error) {
	var GetUserDataResponse *myStartup.GetUserDataResponse

	client, err := skylib.GetRandomClientByProvides("RouteService.RouteGetUserDataRequest")
	if err != nil {
		if GetUserDataResponse == nil {
			GetUserDataResponse = &myStartup.GetUserDataResponse{}
		}
		GetUserDataResponse.Errors = append(GetUserDataResponse.Errors, err.String())
		return GetUserDataResponse, err
	}
	err = client.Call("RouteService.RouteGetUserDataRequest", cr, &GetUserDataResponse)
	if err != nil {
		if GetUserDataResponse == nil {
			GetUserDataResponse = &myStartup.GetUserDataResponse{}

		}
		GetUserDataResponse.Errors = append(GetUserDataResponse.Errors, err.String())
	}

	return GetUserDataResponse, nil
}

// Handler function to accept the submitted form post with the SSN
func submitHandler(env mango.Env) (mango.Status, mango.Headers, mango.Body) {

	log.Println("Submit GetUserData Request")
	cr := &myStartup.GetUserDataRequest{YourInputValue: env.Request().FormValue("YourInputValue")}

	resp, err := submitGetUserDataRequest(cr)
	if err != nil {
		log.Println(err.String())
	}
	log.Println(resp)

	buffer := &bytes.Buffer{}
	respTmpl.Execute(buffer, map[string]interface{}{
		"resp": resp,
	})
	return 200, mango.Headers{}, mango.Body(buffer.String())
}


// Handler function to display the social form
func homeHandler(env mango.Env) (mango.Status, mango.Headers, mango.Body) {
	buffer := &bytes.Buffer{}
	homeTmpl.Execute(buffer, nil)
	return 200, mango.Headers{}, mango.Body(buffer.String())
}

var homeTmpl *template.Template
var respTmpl *template.Template

func main() {
	// Pull in command line options or defaults if none given
	flag.Parse()

	f, err := os.OpenFile(*skylib.LogFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err == nil {
		defer f.Close()
		log.SetOutput(f)
	}

	skylib.Setup(sName)

	homeTmpl = template.MustParse(homeTemplate, nil)
	respTmpl = template.MustParse(responseTemplate, nil)

	rpc.HandleHTTP()

	portString := fmt.Sprintf("%s:%d", *skylib.BindIP, *skylib.Port)

	stack := new(mango.Stack)
	stack.Address = portString

	routes := make(map[string]mango.App)
	routes["/"] = homeHandler
	routes["/new"] = submitHandler
	stack.Middleware(mango.Routing(routes))
	stack.Run(nil)
}

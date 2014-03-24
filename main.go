// HouseMon: process config, env vars, cmdline args, then start up as needed.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/jcw/flow"
	jeebus "github.com/jcw/jeebus/gadgets"
)

var VERSION = "0.9.0" // can be adjusted by goxc at link time
var BUILD_DATE = ""   // can be adjusted by goxc at link time

// defaults can also be overridden through environment variables
const defaults = `
APP_DIR = ./app
BASE_DIR = ./base
DATA_DIR = ./data
HTTP_PORT = :5561
MQTT_PORT = :1883
INIT_FILE =
`

func main() {
	flag.Parse() // required, to set up the proper glog configuration
	flow.LoadConfig(defaults, "./config.txt")
	flow.DontPanic()

	// register more definitions from a JSON-formatted init file, if specified
	if s := flow.Config["INIT_FILE"]; s != "" {
		if err := flow.AddToRegistry(s); err != nil {
			panic(err)
		}
	}

	// if a registered circuit name is given on the command line, run it
	if flag.NArg() > 0 {
		if factory, ok := flow.Registry[flag.Arg(0)]; ok {
			factory().Run()
			return
		}
		fmt.Fprintln(os.Stderr, "Unknown command:", flag.Arg(0))
		os.Exit(1)
	}

	fmt.Printf("Starting webserver for http://localhost:%s/\n",
		flow.Config["HTTP_PORT"])

	// show intro page via a static webserver if the main app dir is absent
	fd, err := os.Open(flow.Config["APP_DIR"])
	if err != nil {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(introPage))
		})
		panic(http.ListenAndServe(":"+flow.Config["HTTP_PORT"], nil))
	}
	fd.Close()

	// normal startup: save config info in database and start the webserver
	c := flow.NewCircuit()

	// database setup, save version and current config settings
	c.Add("db", "LevelDB")
	c.Add("sink", "Sink")
	c.Connect("db.Out", "sink.In", 0)
	c.Connect("db.Mods", "sink.In", 0)
	c.Feed("db.In", flow.Tag{"<clear>", "/config/"})
	for k, v := range flow.Config {
		c.Feed("db.In", flow.Tag{"/config/" + k, v})
	}
	c.Feed("db.In", flow.Tag{"/config/appName", "HouseMon"})
	c.Feed("db.In", flow.Tag{"/config/version", VERSION})
	c.Feed("db.In", flow.Tag{"/config/buildDate", BUILD_DATE})

	// webserver setup
	c.Add("http", "HTTPServer")
	c.Feed("http.Handlers", flow.Tag{"/", flow.Config["APP_DIR"]})
	c.Feed("http.Handlers", flow.Tag{"/base/", flow.Config["BASE_DIR"]})
	c.Feed("http.Handlers", flow.Tag{"/ws", "<websocket>"})

	// start the ball rolling, keep running forever
	c.Add("forever", "Forever")
	c.Run()
}

// introPage contains the HTML shown when the application cannot start normally
const introPage = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>Welcome to HouseMon</title>
  </head>
  <body>
    <blockquote>
      <h3>Welcome to HouseMon</h3>
      <p>Whoops ... the main application files were not found.</p>
      <p>Please launch this application from the HouseMon directory.</p>
    </blockquote>
    <script>
      setInterval(function() {
        ws = new WebSocket("ws://" + location.host + "/ws");
        ws.onopen = function() {
          window.location.reload(true)
        }
      }, 1000)
    </script>
  </body>
</html>`

// Show some additional application information when printing usage info.
func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "HouseMon (%s) + JeeBus (%s) + Flow (%s) %s\n",
			VERSION, jeebus.Version, flow.Version, BUILD_DATE)
		fmt.Fprintln(os.Stderr, "\nDebug options:")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nTry 'help' for more info, or visit %s\n",
			"http://jeelabs.net/projects/housemon/wiki")
	}
}

// This example illustrates how to define a new gadget. It has no input or
// output ports, is registered using a lowercase name, and has a help entry.
// This is only intended for use from the command line, i.e. "housemon info".

func init() {
	flow.Registry["info"] = func() flow.Circuitry { return &infoCmd{} }
	jeebus.Help["info"] = `Show the list of registered gadgets and circuits.`
}

type infoCmd struct{ flow.Gadget }

func (g *infoCmd) Run() {
	fmt.Println("Registered gadgets and circuits:\n")
	flow.PrintRegistry()
	fmt.Println()
}

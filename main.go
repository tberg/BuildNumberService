// vim: sw=2 ts=2 et ai ff=unix fenc=utf-8:

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	// "reflect"
	// "encoding/json"
	"strconv"
	"strings"
	"syscall"
)

var Git = "not set"
var Version = "not set"
var Date = "not set"

type Args struct {
	Config string
}

type Config struct {
	Pidfile      string
	DbPath       string
	VariableName string
	Port         int
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Route functions

type State struct {
	Args Args
	Conf Config
	Db   *sql.DB
}

type BuildNumber struct {
	Build int `json:"BuildNumber" yaml:"BuildNumber"`
}

func (c *State) _addProject(name string) {
	Sql := "insert into state (project, build) values (?, ?)"
	tx, err := c.GetDB().Begin()
	check(err)
	stmt, err := c.Db.Prepare(Sql)
	check(err)
	defer stmt.Close()
	_, err = stmt.Exec(name, 0)
	check(err)
	tx.Commit()
}

func (c *State) _getBuildNumber(name string) int {
	Sql := "select build from state where project=?"
	stmt, err := c.Db.Prepare(Sql)
	check(err)
	defer stmt.Close()
	build := -1
	rows, err := stmt.Query(name)
	check(err)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&build)
		check(err)
		log.Printf("Found build %d for project \"%s\"", build, name)
	}
	if build == -1 {
		log.Printf("New project, creating entry.")
		c._addProject(name)
		build = 0
	}
	return build
}

func (c *State) _setBuildNumber(name string, newval int) {
	Sql := "update state set build=? where project=?"
	tx, err := c.GetDB().Begin()
	check(err)
	stmt, err := c.Db.Prepare(Sql)
	check(err)
	defer stmt.Close()
	_, err = stmt.Exec(newval, name)
	check(err)
	tx.Commit()
}

func (c *State) FormatOutput(build int, style string) string {
	buf := ""
	if strings.Compare(style, "json") == 0 {
		log.Printf("Output style set to JSON")
		buf = fmt.Sprintf("{\"%s\": %d}", c.Conf.VariableName, build)
		// 		data := &BuildNumber{build}
		// 		var err error
		// 		tmp, err := json.Marshal(data)
		// 		check(err)
		// 		buf = string(tmp)
	} else if strings.Compare(style, "yaml") == 0 {
		log.Printf("Output style set to Yaml")
		buf = fmt.Sprintf("%s: %d", c.Conf.VariableName, build)
		// 		data := &BuildNumber{build}
		// 		var err error
		// 		tmp, err := yaml.Marshal(data)
		// 		check(err)
		// 		buf = string(tmp)
	} else {
		// default to bash style
		log.Printf("Output style defaulting to Bash")
		buf = fmt.Sprintf("%s=%d", c.Conf.VariableName, build)
	}
	log.Printf("Output: %s", buf)
	return buf
}

func (c *State) GetBuildNumber(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	_project := params["project"]
	_style := params["style"]
	log.Printf("vars: %s", params)
	log.Printf("project: %s", _project)
	log.Printf("style: %s", _style)

	build := c._getBuildNumber(params["project"])
	log.Printf("GetBuildNumber: %d", build)
	fmt.Fprintf(w, "%s\n", c.FormatOutput(build, _style))
}

func (c *State) IncrementBuildNumber(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	_project := params["project"]
	_style := params["style"]
	log.Printf("vars: %s", params)
	log.Printf("project: %s", _project)
	log.Printf("style: %s", _style)

	build := c._getBuildNumber(_project)
	build += 1
	c._setBuildNumber(_project, build)

	fmt.Fprintf(w, "%s\n", c.FormatOutput(build, _style))
}

func (c *State) SetBuildNumber(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	build, err := strconv.Atoi(params["build"])
	check(err)
	_style := params["style"]
	c._setBuildNumber(params["project"], build)
	log.Printf("Resetting proejct %s to build %d", params["project"], build)
	fmt.Fprintf(w, "%s\n", c.FormatOutput(build, _style))
}

func (c *State) GetPortString() string {
	return fmt.Sprintf(":%d", c.Conf.Port)
}

func (c *State) GetDB() *sql.DB {
	if nil == c.Db {
		tmp, err := sql.Open("sqlite3", c.Conf.DbPath)
		check(err)
		c.Db = tmp
		log.Printf("Assigned to db")
	}
	return c.Db
}

func (c *State) ParseArgs() {
	c.Args = Args{}
	flag.StringVar(&c.Args.Config, "config", "/etc/bns.yaml", "Alternate configuration file.")
	flag.Parse()
}

func (c *State) LoadConfig() {
	c.Conf = Config{}
	dat, err := ioutil.ReadFile(c.Args.Config)
	check(err)
	err = yaml.Unmarshal(dat, &c.Conf)
	check(err)
}

func (c *State) WritePidfile() {
	pidbuf := fmt.Sprintf("%d\n", os.Getpid())
	err := ioutil.WriteFile(c.Conf.Pidfile, []byte(pidbuf), 644)
	check(err)
}

func (c *State) CreateExitHandler() {
	sigChannel := make(chan os.Signal, 2)
	signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChannel
		// cleanup(c.Conf.Pidfile)
		c.Close()
		os.Exit(1)
	}()
}

func (c *State) CreateDB() {
	create := `
  create table if not exists state (
    project varchar(30) primary key not null,
    build integer
  );
  `
	_, err := c.GetDB().Exec(create)
	check(err)
}

func (c *State) Close() {
	log.Printf("Closing.")
	c.Db.Close()
}

func (c *State) CreateRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/{project:.*?}/inc/{style:(?:bash|json|yaml)}", c.IncrementBuildNumber).Methods("GET")
	router.HandleFunc("/{project:.*}/inc", c.IncrementBuildNumber).Methods("GET")
	router.HandleFunc("/{project:.*?}/{style:(?:bash|json|yaml)}", c.GetBuildNumber).Methods("GET")
	router.HandleFunc("/{project:.*}", c.GetBuildNumber).Methods("GET")
	router.HandleFunc("/{project:.*?}/{build}", c.SetBuildNumber).Methods("POST")
	return router
}

func (c *State) Run() {
	c.ParseArgs()
	c.LoadConfig()
	c.WritePidfile()
	c.CreateExitHandler()
	c.CreateDB()
	defer c.Close()
	log.Fatal(http.ListenAndServe(c.GetPortString(), c.CreateRouter()))
}

////////////////////////////////////////////////////////////////////////////////
//Entrypoint

func main() {
	log.Printf("Build Number Server version %s starting up", Version)
	log.Printf(Git)

	state := State{}
	state.Run()

}

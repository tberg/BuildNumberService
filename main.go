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
	"encoding/json"
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
	Pidfile string
	DbPath  string
	Port    int
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func parse_args() Args {
	args := Args{}
	flag.StringVar(&args.Config, "config", "/etc/bns.yaml", "Alternate configuration file.")
	flag.Parse()
	return args
}

func load_config(path string) Config {
	conf := Config{}
	dat, err := ioutil.ReadFile(path)
	check(err)
	err = yaml.Unmarshal(dat, &conf)
	check(err)
	return conf
	/*
	  conf := Config{"/var/run/my.pid", "/var/lib/bns/"}
	  buf, err := yaml.Marshal(conf)
	  check(err)
	  fmt.Printf("conf: %s", buf)
	  err = ioutil.WriteFile("test.yaml", buf, 644)
	  check(err)
	*/
}

func cleanup(pidfile string) {
	err := os.Remove(pidfile)
	if nil != err {
		log.Fatal(err)
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
		data := &BuildNumber{build}
		var err error
		tmp, err := json.Marshal(data)
		check(err)
		buf = string(tmp)
	} else if strings.Compare(style, "yaml") == 0 {
		log.Printf("Output style set to Yaml")
		data := &BuildNumber{build}
		var err error
		tmp, err := yaml.Marshal(data)
		check(err)
		buf = string(tmp)
	} else {
		// default to bash style
		log.Printf("Output style defaulting to Bash")
		buf = fmt.Sprintf("SS_BUILD_NUMBER=%d", build)
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
	c._setBuildNumber(params["project"], build)
	log.Printf("Resetting proejct %s to build %d", params["project"], build)
	fmt.Fprintf(w, "SS_BUILD_NUMBER=%d\n", build)
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

////////////////////////////////////////////////////////////////////////////////
//Entrypoint

func main() {
	log.Printf("Build Number Server version %s starting up", Version)
	log.Printf(Git)
	// Parse command line
	args := parse_args()
	fmt.Printf("Config file: %s\n", args.Config)

	// load configuration
	conf := load_config(args.Config)
	fmt.Printf("pidfile: %s\n", conf.Pidfile)
	fmt.Printf("dbpath: %s\n", conf.DbPath)
	fmt.Printf("port: %d\n", conf.Port)

	// Create pidfile
	pidbuf := fmt.Sprintf("%d\n", os.Getpid())
	err := ioutil.WriteFile(conf.Pidfile, []byte(pidbuf), 644)
	check(err)

	// set cleanup hook
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cleanup(conf.Pidfile)
		os.Exit(1)
	}()

	state := State{args, conf, nil}

	create := `
  create table if not exists state (
    project varchar(30) primary key not null,
    build integer
  );
  `
	// 	_, err = db.Exec(create)
	// 	check(err)
	_, err = state.GetDB().Exec(create)
	check(err)
	defer state.GetDB().Close()

	// buf := fmt.Sprintf(":%d", conf.Port)
	router := mux.NewRouter().StrictSlash(true)
	// router.HandleFunc("/{project}", state.GetBuildNumber).Methods("GET").Queries("action", "{new}")
	router.HandleFunc("/{project}/inc", state.IncrementBuildNumber).Methods("GET")
	router.HandleFunc("/{project}/inc/{style:(?:bash|json|yaml)}", state.IncrementBuildNumber).Methods("GET")
	router.HandleFunc("/{project}", state.GetBuildNumber).Methods("GET")
	router.HandleFunc("/{project}/{style:(?:bash|json|yaml)}", state.GetBuildNumber).Methods("GET")
	router.HandleFunc("/{project}/{build}", state.SetBuildNumber).Methods("POST")
	log.Fatal(http.ListenAndServe(state.GetPortString(), router))
}

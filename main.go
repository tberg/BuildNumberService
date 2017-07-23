// vim: sw=2 ts=2 et ai ff=unix fenc=utf-8:

package main

import "flag"
import "fmt"
import "gopkg.in/yaml.v2"
import "io/ioutil"

import "github.com/mattn/go-sqlite3"
import "os"
import "os/signal"
import "syscall"

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

func cleanup() {
	fmt.Println("goodbye.")
}

func main() {
	// set cleanup hook
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cleanup()
		os.Exit(1)
	}()

	// Parse command line
	args := parse_args()
	fmt.Printf("Config file: %s\n", args.Config)

	// load configuration
	conf := load_config(args.Config)
	fmt.Printf("pidfile: %s\n", conf.Pidfile)
	fmt.Printf("dbpath: %s\n", conf.DbPath)
	fmt.Printf("port: %d\n", conf.Port)

	// Set up REST api routes
	db, err := sql.Open(conf.DbPath)
	check(err)

}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server struct {
		Host    string `yaml:"host"`
		Port    string `yaml:"port"`
		Timeout struct {
			Server time.Duration `yaml:"server"`
			Read   time.Duration `yaml:"read"`
			Write  time.Duration `yaml:"write"`
			Idle   time.Duration `yaml:"idle"`
		} `yaml:"timeout"`
	} `yaml:"server"`
}

func ParseFlags() (string, error) {
	var configPath string

	flag.StringVar(&configPath, "config", "./config.yaml", "path to config file")

	flag.Parse()

	if err := ValidateConfigPath(configPath); err != nil {
		return "", err
	}

	return configPath, nil
}

func ValidateConfigPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("%s is a directory, not a file", path)
	}

	return nil
}

func NewConfig(configPath string) (*Config, error) {
	config := &Config{}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)
	if err := d.Decode(&config); err != nil {
		return nil, err
	}
	return config, nil
}

func NewRouter() *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, you've requested %s\n", r.URL.Path)
	})

	return router
}

func (config Config) Run() {
	var runChan = make(chan os.Signal, 1)

	ctx, cancel := context.WithTimeout(context.Background(), config.Server.Timeout.Server)
	defer cancel()

	server := &http.Server{
		Addr:         config.Server.Host + ":" + config.Server.Port,
		Handler:      NewRouter(),
		ReadTimeout:  config.Server.Timeout.Read * time.Second,
		WriteTimeout: config.Server.Timeout.Write * time.Second,
		IdleTimeout:  config.Server.Timeout.Idle * time.Second,
	}

	signal.Notify(runChan, os.Interrupt)

	log.Printf("Server is starting on %s\n", server.Addr)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {

			} else {
				log.Fatalf("Server failed to start due to %s", err)
			}
		}
	}()

	interrupt := <-runChan

	log.Printf("Server is shutting down due to %+v", interrupt)

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server was unable to gracefully shutdown due to %+v", err)
	}
}

func main() {
	cfgPath, err := ParseFlags()
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := NewConfig(cfgPath)
	if err != nil {
		log.Fatal(err)
	}

	cfg.Run()
}

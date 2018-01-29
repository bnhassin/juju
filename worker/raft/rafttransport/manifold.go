// Copyright 2018 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package rafttransport

import (
	"github.com/hashicorp/raft"
	"github.com/juju/errors"
	"gopkg.in/juju/worker.v1"

	"github.com/juju/juju/agent"
	"github.com/juju/juju/api"
	"github.com/juju/juju/apiserver/apiserverhttp"
	"github.com/juju/juju/worker/dependency"
)

// ManifoldConfig holds the information necessary to run an apiserver-based
// raft transport worker in a dependency.Engine.
type ManifoldConfig struct {
	AgentName string
	MuxName   string

	DialConn  DialConnFunc
	NewWorker func(Config) (worker.Worker, error)

	// Path is the path of the raft HTTP endpoint.
	Path string
}

// Validate validates the manifold configuration.
func (config ManifoldConfig) Validate() error {
	if config.AgentName == "" {
		return errors.NotValidf("empty AgentName")
	}
	if config.MuxName == "" {
		return errors.NotValidf("empty MuxName")
	}
	if config.DialConn == nil {
		return errors.NotValidf("nil DialConn")
	}
	if config.NewWorker == nil {
		return errors.NotValidf("nil NewWorker")
	}
	if config.Path == "" {
		return errors.NotValidf("empty Path")
	}
	return nil
}

// Manifold returns a dependency.Manifold that will run an apiserver-based
// raft transport worker.
func Manifold(config ManifoldConfig) dependency.Manifold {
	return dependency.Manifold{
		Inputs: []string{
			config.AgentName,
			config.MuxName,
		},
		Start:  config.start,
		Output: transportOutput,
	}
}

// start is a method on ManifoldConfig because it's more readable than a closure.
func (config ManifoldConfig) start(context dependency.Context) (worker.Worker, error) {
	if err := config.Validate(); err != nil {
		return nil, errors.Trace(err)
	}

	var agent agent.Agent
	if err := context.Get(config.AgentName, &agent); err != nil {
		return nil, errors.Trace(err)
	}

	var mux *apiserverhttp.Mux
	if err := context.Get(config.MuxName, &mux); err != nil {
		return nil, errors.Trace(err)
	}

	apiInfo, ok := agent.CurrentConfig().APIInfo()
	if !ok {
		return nil, dependency.ErrMissing
	}
	certPool, err := api.CreateCertPool(apiInfo.CACert)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return config.NewWorker(Config{
		APIInfo:   apiInfo,
		DialConn:  config.DialConn,
		Mux:       mux,
		Path:      config.Path,
		Tag:       agent.CurrentConfig().Tag(),
		TLSConfig: api.NewTLSConfig(certPool),
	})
}

func transportOutput(in worker.Worker, out interface{}) error {
	t, ok := in.(raft.Transport)
	if !ok {
		return errors.Errorf("expected input of type %T, got %T", t, in)
	}
	tout, ok := out.(*raft.Transport)
	if ok {
		*tout = t
		return nil
	}
	return errors.Errorf("expected output of type %T, got %T", tout, out)
}

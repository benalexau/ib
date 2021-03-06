package ib

import (
	"errors"
	"flag"
	"os"
	"reflect"
	"testing"
	"time"
)

func (engine *Engine) expect(t *testing.T, seconds int, ch chan Reply, expected []IncomingMessageId) (Reply, error) {
	for {
		select {
		case <-time.After(time.Duration(seconds) * time.Second):
			return nil, errors.New("Timeout waiting")
		case v := <-ch:
			if v.code() == 0 {
				t.Fatalf("don't know message '%v'", v)
			}
			for _, code := range expected {
				if v.code() == code {
					return v, nil
				}
			}
			// wrong message received
			t.Logf("received message '%v' of type '%v'\n",
				v, reflect.ValueOf(v).Type())
		}
	}

	return nil, nil
}

// private variable for mantaining engine reuse in test
// use TestEngine instead of this
var testEngine *Engine
var noEngineReuse = flag.Bool("no-engine-reuse", false,
	"Don't keep reusing the engine; each test case gets its own engine.")

// Engine for test reuse.
//
// Unless the test runner is passed the -no-engine-reuse flag, this will keep
// reusing the same engine.
func NewTestEngine(t *testing.T) *Engine {

	if testEngine == nil {
		opts := NewEngineOptions{Gateway: "127.0.0.1:4002"}
		if os.Getenv("CI") != "" || os.Getenv("IB_ENGINE_DUMP") != "" {
			opts.DumpConversation = true
		}
		engine, err := NewEngine(opts)

		if err != nil {
			t.Fatalf("cannot connect engine: %s", err)
		}

		if *noEngineReuse {
			t.Log("created new engine, no reuse")
			return engine
		} else {
			t.Log("created engine for reuse")
			testEngine = engine
			return engine
		}
	}

	if testEngine.State() != EngineReady {
		t.Fatalf("engine (client ID %d) not ready (did a prior test Stop() rather than ConditionalStop() ?)", testEngine.client)
	}

	t.Log("reusing engine; state: %v", testEngine.State())
	return testEngine
}

// Will actually do a stop only if the flag -no-engine-reuse is active
func (e *Engine) ConditionalStop(t *testing.T) {
	if *noEngineReuse {
		t.Log("no engine reuse, stopping engine")
		e.Stop()
		t.Log("engine state: %v", e.State())
	}
}

func TestConnect(t *testing.T) {
	opts := NewEngineOptions{Gateway: "127.0.0.1:4002"}
	if os.Getenv("CI") != "" || os.Getenv("IB_ENGINE_DUMP") != "" {
		opts.DumpConversation = true
	}
	engine, err := NewEngine(opts)

	if err != nil {
		t.Fatalf("cannot connect engine: %s", err)
	}

	defer engine.Stop()

	if engine.State() != EngineReady {
		t.Fatalf("engine is not ready")
	}

	if engine.serverTime.IsZero() {
		t.Fatalf("server time not provided")
	}

	var states chan EngineState = make(chan EngineState)
	engine.SubscribeState(states)

	// stop the engine in 100 ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		engine.Stop()
	}()

	newState := <-states

	if newState != EngineExitNormal {
		t.Fatalf("engine state change error")
	}

	err = engine.FatalError()
	if err != nil {
		t.Fatalf("engine reported an error: %v", err)
	}
}

func logreply(t *testing.T, reply Reply, err error) {
	if reply == nil {
		t.Logf("received reply nil")
	} else {
		t.Logf("received reply '%v' of type %v", reply, reflect.ValueOf(reply).Type())
	}
	if err != nil {
		t.Logf(" (error: '%v')", err)
	}
	t.Logf("\n")
}

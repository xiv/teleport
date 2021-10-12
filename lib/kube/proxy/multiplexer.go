/*
Copyright 2021 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package proxy

import (
	"io"
	"sync"
)

type Multiplexer struct {
	mu sync.Mutex
	clientsModified chan struct{}
	stdinR []io.Reader
	stdoutW []io.Writer
	stderrW []io.Writer
}

func NewMultiplexer() *Multiplexer {
	clientsModified := make(chan struct{})
	clientsModified <- struct{}{}

	return &Multiplexer{
		clientsModified: clientsModified,
	}
}

func (m* Multiplexer) RegisterClient(stdin io.Reader, stdout io.Writer, stderr io.Writer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stdinR = append(m.stdinR, stdin)
	m.stdoutW = append(m.stdoutW, stdout)
	m.stderrW = append(m.stderrW, stderr)

	m.clientsModified <- struct{}{}
}

func (m* Multiplexer) Run(stdin io.Writer, stdout io.Reader, stderr io.Reader) {
	var stdinR io.Reader
	var stdoutW io.Writer
	var stderrW io.Writer
	notifier := make(chan struct{})

	for {
		_, _ = <- m.clientsModified
		m.mu.Lock()
		stdinR = io.MultiReader(m.stdinR...)
		stdoutW = io.MultiWriter(m.stdoutW...)
		stderrW = io.MultiWriter(m.stderrW...)
		m.mu.Unlock()

		close(notifier)
		notifier = make(chan struct{})

		copyUntil(stdinR, stdin, notifier)
		copyUntil(stdout, stdoutW, notifier)
		copyUntil(stderr, stderrW, notifier)
	}
}

func copyUntil(src io.Reader, dst io.Writer, notifier chan struct{}) {

}
package engine

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/aquasecurity/tracee/types/detect"
	"github.com/aquasecurity/tracee/types/protocol"
	"github.com/aquasecurity/tracee/types/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type regoFakeSignature struct {
	getMetadata       func() (detect.SignatureMetadata, error)
	getSelectedEvents func() ([]detect.SignatureEventSelector, error)
	init              func(detect.SignatureHandler) error
	onEvent           func(protocol.Event) error
	onSignal          func(signal detect.Signal) error
}

func (fs regoFakeSignature) GetMetadata() (detect.SignatureMetadata, error) {
	if fs.getMetadata != nil {
		return fs.getMetadata()
	}

	return detect.SignatureMetadata{
		Name: "Fake Signature",
	}, nil
}

func (fs regoFakeSignature) GetSelectedEvents() ([]detect.SignatureEventSelector, error) {
	if fs.getSelectedEvents != nil {
		return fs.getSelectedEvents()
	}

	return []detect.SignatureEventSelector{}, nil
}

func (fs regoFakeSignature) Init(cb detect.SignatureHandler) error {
	if fs.init != nil {
		return fs.init(cb)
	}
	return nil
}

func (fs regoFakeSignature) OnEvent(event protocol.Event) error {
	if fs.onEvent != nil {
		return fs.onEvent(event)
	}
	return nil
}

func (fs regoFakeSignature) OnSignal(signal detect.Signal) error {
	if fs.onSignal != nil {
		return fs.onSignal(signal)
	}
	return nil
}
func (fs *regoFakeSignature) Close() {}

func TestConsumeSources(t *testing.T) {
	testCases := []struct {
		name              string
		inputEvent        trace.Event
		inputSignature    regoFakeSignature
		expectedNumEvents int
		expectedError     string
		expectedEvent     interface{}
		config            Config
	}{
		{
			name: "happy path - with one matching selector, parsed event enabled",
			inputEvent: trace.Event{
				EventName:       "test_event",
				ProcessID:       2,
				ParentProcessID: 1,
				Args: []trace.Argument{
					{
						ArgMeta: trace.ArgMeta{
							Name: "pathname",
						},
						Value: "/proc/self/mem",
					},
				},
			},
			inputSignature: regoFakeSignature{
				getSelectedEvents: func() ([]detect.SignatureEventSelector, error) {
					return []detect.SignatureEventSelector{
						{
							Name:   "test_event",
							Source: "tracee",
						},
					}, nil
				},
			},
			expectedNumEvents: 1,
			expectedEvent: trace.Event{
				ProcessID: 2, ParentProcessID: 1, Args: []trace.Argument{{ArgMeta: trace.ArgMeta{Name: "pathname", Type: ""}, Value: "/proc/self/mem"}},
				EventName: "test_event",
			},
			config: Config{},
		},
		{
			name: "happy path - with one matching selector",
			inputEvent: trace.Event{
				EventName:       "test_event",
				ProcessID:       2,
				ParentProcessID: 1,
				Args: []trace.Argument{
					{
						ArgMeta: trace.ArgMeta{
							Name: "pathname",
						},
						Value: "/proc/self/mem",
					},
				},
			},
			inputSignature: regoFakeSignature{
				getSelectedEvents: func() ([]detect.SignatureEventSelector, error) {
					return []detect.SignatureEventSelector{
						{
							Name:   "test_event",
							Source: "tracee",
						},
					}, nil
				},
			},
			expectedNumEvents: 1,
			expectedEvent: trace.Event{
				ProcessID: 2, ParentProcessID: 1, Args: []trace.Argument{{ArgMeta: trace.ArgMeta{Name: "pathname", Type: ""}, Value: "/proc/self/mem"}},
				EventName: "test_event",
			},
		},
		{
			name: "happy path - with no matching event selector",
			inputEvent: trace.Event{
				EventName: "execve",
			},
			inputSignature: regoFakeSignature{
				getSelectedEvents: func() ([]detect.SignatureEventSelector, error) {
					return []detect.SignatureEventSelector{
						{
							Name:   "not execve",
							Source: "tracee",
						},
					}, nil
				},
			},
			expectedNumEvents: 0,
		},
		{
			name: "happy path - with all events selector",
			inputEvent: trace.Event{
				EventName: "execve",
			},
			inputSignature: regoFakeSignature{
				getSelectedEvents: func() ([]detect.SignatureEventSelector, error) {
					return []detect.SignatureEventSelector{
						{
							Name:   "*",
							Source: "tracee",
						},
					}, nil
				},
			},
			expectedNumEvents: 1,
			expectedEvent: trace.Event{
				EventName: "execve",
			},
		},
		{
			name:       "happy path - with all events selector, no name",
			inputEvent: trace.Event{EventName: "execve"},
			inputSignature: regoFakeSignature{
				getSelectedEvents: func() ([]detect.SignatureEventSelector, error) {
					return []detect.SignatureEventSelector{
						{
							Source: "tracee",
						},
					}, nil
				},
			},
			expectedNumEvents: 1,
			expectedEvent:     trace.Event{EventName: "execve"},
		},
		{
			name: "happy path - with one matching selector including event origin from container",
			inputEvent: trace.Event{
				EventName:       "test_event",
				ProcessID:       2,
				ParentProcessID: 1,
				ContainerID:     "container ID",
				Args: []trace.Argument{
					{
						ArgMeta: trace.ArgMeta{
							Name: "pathname",
						},
						Value: "/proc/self/mem",
					},
				},
			},
			inputSignature: regoFakeSignature{
				getSelectedEvents: func() ([]detect.SignatureEventSelector, error) {
					return []detect.SignatureEventSelector{
						{
							Name:   "test_event",
							Source: "tracee",
							Origin: "container",
						},
					}, nil
				},
			},
			expectedNumEvents: 1,
			expectedEvent: trace.Event{
				ProcessID: 2, ParentProcessID: 1, ContainerID: "container ID", Args: []trace.Argument{{ArgMeta: trace.ArgMeta{Name: "pathname", Type: ""}, Value: "/proc/self/mem"}},
				EventName: "test_event",
			},
		},
		{
			name: "happy path - with one matching selector with mismatching event origin from container",
			inputEvent: trace.Event{
				EventName:       "test_event",
				ProcessID:       2,
				ParentProcessID: 1,
				ContainerID:     "container ID",
				Args: []trace.Argument{
					{
						ArgMeta: trace.ArgMeta{
							Name: "pathname",
						},
						Value: "/proc/self/mem",
					},
				},
			},
			inputSignature: regoFakeSignature{
				getSelectedEvents: func() ([]detect.SignatureEventSelector, error) {
					return []detect.SignatureEventSelector{
						{
							Name:   "test_event",
							Source: "tracee",
							Origin: "host",
						},
					}, nil
				},
			},
			expectedNumEvents: 0,
		},
		{
			name: "happy path - with one matching selector including event origin from host",
			inputEvent: trace.Event{
				EventName:       "test_event",
				ProcessID:       2,
				ParentProcessID: 2,
				Args: []trace.Argument{
					{
						ArgMeta: trace.ArgMeta{
							Name: "pathname",
						},
						Value: "/proc/self/mem",
					},
				},
			},
			inputSignature: regoFakeSignature{
				getSelectedEvents: func() ([]detect.SignatureEventSelector, error) {
					return []detect.SignatureEventSelector{
						{
							Name:   "test_event",
							Source: "tracee",
							Origin: "container",
						},
					}, nil
				},
			},
			expectedNumEvents: 1,
			expectedEvent: trace.Event{
				ProcessID: 2, ParentProcessID: 2, Args: []trace.Argument{{ArgMeta: trace.ArgMeta{Name: "pathname", Type: ""}, Value: "/proc/self/mem"}},
				EventName: "test_event",
			},
		},
		{
			name: "sad path - with all events selector, no source",
			inputSignature: regoFakeSignature{
				getSelectedEvents: func() ([]detect.SignatureEventSelector, error) {
					return []detect.SignatureEventSelector{
						{
							Name: "*",
						},
					}, nil
				},
			},
			expectedError: "signature Fake Signature doesn't declare an input source\n",
		},
		{
			name: "sad path - signature init fails",
			inputSignature: regoFakeSignature{
				init: func(handler detect.SignatureHandler) error {
					return errors.New("init failed")
				},
			},
			expectedNumEvents: 0,
			expectedError:     "error initializing signature Fake Signature: init failed\n",
		},
		{
			name: "sad path - getSelectedEvents returns an error",
			inputSignature: regoFakeSignature{
				getSelectedEvents: func() ([]detect.SignatureEventSelector, error) {
					return nil, errors.New("getSelectedEvents error")
				},
			},
			expectedError: "error getting selected events for signature Fake Signature: getSelectedEvents error\n",
		},
		{
			name: "sad path - getMetadata returns an error",
			inputSignature: regoFakeSignature{
				getMetadata: func() (detect.SignatureMetadata, error) {
					return detect.SignatureMetadata{}, errors.New("getMetadata error")
				},
			},
			expectedError: "error getting metadata: getMetadata error\n",
		},
		{
			name: "sad path - event ContainerID was not parsed but event is from container",
			inputEvent: trace.Event{
				EventName:       "test_event",
				ProcessID:       2,
				ParentProcessID: 1,
				Args: []trace.Argument{
					{
						ArgMeta: trace.ArgMeta{
							Name: "pathname",
						},
						Value: "/proc/self/mem",
					},
				},
			},
			inputSignature: regoFakeSignature{
				getSelectedEvents: func() ([]detect.SignatureEventSelector, error) {
					return []detect.SignatureEventSelector{
						{
							Name:   "test_event",
							Source: "tracee",
							Origin: "container",
						},
					}, nil
				},
			},
			expectedNumEvents: 1,
			expectedEvent: trace.Event{
				ProcessID: 2, ParentProcessID: 1, Args: []trace.Argument{{ArgMeta: trace.ArgMeta{Name: "pathname", Type: ""}, Value: "/proc/self/mem"}},
				EventName: "test_event",
			},
		},
	}

	emptyEvent := protocol.Event{}

	for _, tc := range testCases {
		inputs := EventSources{}
		inputs.Tracee = make(chan protocol.Event, 1)
		outputChan := make(chan detect.Finding, 1)
		done := make(chan bool, 1)
		var logBuf []byte
		logger := bytes.NewBuffer(logBuf)

		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				// signal the end
				done <- true

				// cleanup
				close(done)
				close(outputChan)
				close(inputs.Tracee)
			}()

			var sigs []detect.Signature
			sigs = append(sigs, &tc.inputSignature)

			var gotNumEvents int
			tc.inputSignature.onEvent = func(event protocol.Event) error {
				assert.Equal(t, tc.expectedEvent, event.Payload.(trace.Event), tc.name)
				gotNumEvents++
				return nil
			}

			e, err := NewEngine(sigs, inputs, outputChan, logger, tc.config)
			require.NoError(t, err, "constructing engine")
			go func() {
				e.Start(done)
			}()

			// send a test event
			e.inputs.Tracee <- tc.inputEvent.ToProtocol()

			// assert
			var gotEvent protocol.Event
			time.Sleep(time.Millisecond * 1) // wait for events to propagate

			if tc.expectedNumEvents <= 0 {
				assert.Equal(t, emptyEvent, gotEvent, tc.name)
				assert.Zero(t, gotNumEvents, tc.name)
			} else {
				assert.Equal(t, tc.expectedNumEvents, gotNumEvents, tc.name)
			}

			if tc.expectedError != "" {
				assert.Contains(t, logger.String(), tc.expectedError, tc.name)
			}
		})
	}
}

func TestEventSignatureSelector(t *testing.T) {
	notTraceeEvt := protocol.Event{
		Headers: protocol.EventHeaders{
			ContentType: "tracee.notevent.lol",
			Origin:      "nottracee/*",
		},
		Payload: "just some stuff",
	}
	_, err := eventSignatureSelector(notTraceeEvt)

	require.Error(t, err)

	testCases := []trace.Event{
		{
			EventName:       "test_event",
			ProcessID:       2,
			HostProcessID:   2,
			ParentProcessID: 1,
			Args: []trace.Argument{
				{
					ArgMeta: trace.ArgMeta{
						Name: "pathname",
					},
					Value: "/proc/self/mem",
				},
			},
		},
		{
			EventName: "execve",
		},
	}

	for _, traceeEvt := range testCases {
		expected := detect.SignatureEventSelector{
			Origin: string(trace.HostOrigin),
			Name:   traceeEvt.EventName,
			Source: "tracee",
		}

		selector, _ := eventSignatureSelector(traceeEvt.ToProtocol())

		assert.Equal(t, expected, selector)
	}
}

func TestGetSelectedEvents(t *testing.T) {
	sigs := []detect.Signature{
		&regoFakeSignature{
			getSelectedEvents: func() ([]detect.SignatureEventSelector, error) {
				return []detect.SignatureEventSelector{
					{
						Name:   "test_event",
						Source: "tracee",
					},
					{
						Name:   "test_event2",
						Source: "tracee",
					},
				}, nil
			},
		},
		&regoFakeSignature{
			getSelectedEvents: func() ([]detect.SignatureEventSelector, error) {
				return []detect.SignatureEventSelector{
					{
						Name:   "test_event",
						Source: "tracee",
						Origin: "host",
					},
					{
						Name:   "test_event2",
						Source: "tracee",
					},
				}, nil
			},
		},
	}
	e, err := NewEngine(sigs, EventSources{Tracee: make(chan protocol.Event)}, make(chan detect.Finding), &bytes.Buffer{}, Config{})
	require.NoError(t, err, "constructing engine")
	se := e.GetSelectedEvents()
	expected := []detect.SignatureEventSelector{
		{
			Name:   "test_event",
			Source: "tracee",
			Origin: "*",
		},
		{
			Name:   "test_event2",
			Source: "tracee",
			Origin: "*",
		},
		{
			Name:   "test_event",
			Source: "tracee",
			Origin: "host",
		},
	}
	assert.ElementsMatch(t, expected, se)
}

package desktop

import (
	"context"
	"io"
	"testing"

	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/api/types/events"
	libevents "github.com/gravitational/teleport/lib/events"
	"github.com/gravitational/teleport/lib/tlsca"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func setup() (*WindowsService, *tlsca.Identity, *libevents.MockEmitter) {
	emitter := &libevents.MockEmitter{}
	log := logrus.New()
	log.SetOutput(io.Discard)

	s := &WindowsService{
		clusterName: "test-cluster",
		cfg: WindowsServiceConfig{
			Log:     log,
			Emitter: emitter,
			Heartbeat: HeartbeatConfig{
				HostUUID: "test-host-id",
			},
		},
	}

	id := &tlsca.Identity{
		Username:     "foo",
		Impersonator: "bar",
		MFAVerified:  "mfa-id",
		ClientIP:     "127.0.0.1",
	}

	return s, id, emitter
}

func TestSessionStartEvent(t *testing.T) {
	s, id, emitter := setup()

	desktop := &types.WindowsDesktopV3{
		ResourceHeader: types.ResourceHeader{
			Metadata: types.Metadata{
				Name:   "test-desktop",
				Labels: map[string]string{"env": "production"},
			},
		},
		Spec: types.WindowsDesktopSpecV3{
			Addr:   "192.168.100.12",
			Domain: "test.example.com",
		},
	}

	s.onSessionStart(
		context.Background(),
		id,
		"Administrator",
		"sessionID",
		desktop,
		true,
	)

	event := emitter.LastEvent()
	require.NotNil(t, event)

	startEvent, ok := event.(*events.WindowsDesktopSessionStart)
	require.True(t, ok)

	require.Equal(t, libevents.WindowsDesktopSessionStartEvent, startEvent.Type)
	require.Equal(t, libevents.DesktopSessionStartCode, startEvent.Code)
	require.Equal(t, s.clusterName, startEvent.ClusterName)
	require.Equal(t, id.Username, startEvent.User)
	require.Equal(t, id.Impersonator, startEvent.Impersonator)
	require.Equal(t, id.MFAVerified, startEvent.WithMFA)
	require.Equal(t, id.ClientIP, startEvent.LocalAddr)
	require.Equal(t, desktop.GetAddr(), startEvent.RemoteAddr)
	require.Equal(t, libevents.EventProtocolRDP, startEvent.Protocol)
	require.True(t, startEvent.Success)
	require.Equal(t, s.cfg.Heartbeat.HostUUID, startEvent.WindowsDesktopService)
	require.Equal(t, desktop.GetAddr(), startEvent.DesktopAddr)
	require.Equal(t, desktop.GetDomain(), startEvent.Domain)
	require.Equal(t, "Administrator", startEvent.WindowsUser)
	require.Equal(t, "sessionID", startEvent.SessionID)
	require.Equal(t, map[string]string{"env": "production"}, startEvent.DesktopLabels)
}

func TestSessionEndEvent(t *testing.T) {
	s, id, emitter := setup()

	desktop := &types.WindowsDesktopV3{
		ResourceHeader: types.ResourceHeader{
			Metadata: types.Metadata{
				Name:   "test-desktop",
				Labels: map[string]string{"env": "production"},
			},
		},
		Spec: types.WindowsDesktopSpecV3{
			Addr:   "192.168.100.12",
			Domain: "test.example.com",
		},
	}

	s.onSessionEnd(
		context.Background(),
		id,
		"Administrator",
		"sessionID",
		desktop,
	)

	event := emitter.LastEvent()
	require.NotNil(t, event)
	endEvent, ok := event.(*events.WindowsDesktopSessionEnd)
	require.True(t, ok)

	require.Equal(t, libevents.WindowsDesktopSessionEndEvent, endEvent.Type)
	require.Equal(t, libevents.DesktopSessionEndCode, endEvent.Code)
	require.Equal(t, s.clusterName, endEvent.ClusterName)
	require.Equal(t, id.Username, endEvent.User)
	require.Equal(t, id.Impersonator, endEvent.Impersonator)
	require.Equal(t, id.MFAVerified, endEvent.WithMFA)
	require.Equal(t, s.cfg.Heartbeat.HostUUID, endEvent.WindowsDesktopService)
	require.Equal(t, desktop.GetAddr(), endEvent.DesktopAddr)
	require.Equal(t, desktop.GetDomain(), endEvent.Domain)
	require.Equal(t, "Administrator", endEvent.WindowsUser)
	require.Equal(t, "sessionID", endEvent.SessionID)
	require.Equal(t, map[string]string{"env": "production"}, endEvent.DesktopLabels)
}

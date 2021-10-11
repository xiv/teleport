package desktop

import (
	"context"

	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/api/types/events"
	libevents "github.com/gravitational/teleport/lib/events"
	"github.com/gravitational/teleport/lib/tlsca"
)

func (s *WindowsService) onSessionStart(ctx context.Context, id *tlsca.Identity, windowsUser, sessionID string, desktop types.WindowsDesktop, success bool) {
	event := &events.WindowsDesktopSessionStart{
		Metadata: events.Metadata{
			Type:        libevents.WindowsDesktopSessionStartEvent,
			Code:        libevents.DesktopSessionStartCode,
			ClusterName: s.clusterName,
		},
		UserMetadata: events.UserMetadata{
			User:         id.Username,
			Impersonator: id.Impersonator,
		},
		SessionMetadata: events.SessionMetadata{
			SessionID: sessionID,
			WithMFA:   id.MFAVerified,
		},
		ConnectionMetadata: events.ConnectionMetadata{
			LocalAddr:  id.ClientIP,
			RemoteAddr: desktop.GetAddr(),
			Protocol:   libevents.EventProtocolRDP,
		},
		Status: events.Status{
			Success: success,
		},
		WindowsDesktopService: s.cfg.Heartbeat.HostUUID,
		DesktopAddr:           desktop.GetAddr(),
		Domain:                desktop.GetDomain(),
		WindowsUser:           windowsUser,
		DesktopLabels:         desktop.GetAllLabels(),
	}
	s.emit(ctx, event)
}

func (s *WindowsService) onSessionEnd(ctx context.Context, id *tlsca.Identity, windowsUser, sessionID string, desktop types.WindowsDesktop) {
	event := &events.WindowsDesktopSessionEnd{
		Metadata: events.Metadata{
			Type:        libevents.WindowsDesktopSessionEndEvent,
			Code:        libevents.DesktopSessionEndCode,
			ClusterName: s.clusterName,
		},
		UserMetadata: events.UserMetadata{
			User:         id.Username,
			Impersonator: id.Impersonator,
		},
		SessionMetadata: events.SessionMetadata{
			SessionID: sessionID,
			WithMFA:   id.MFAVerified,
		},
		WindowsDesktopService: s.cfg.Heartbeat.HostUUID,
		DesktopAddr:           desktop.GetAddr(),
		Domain:                desktop.GetDomain(),
		WindowsUser:           windowsUser,
		DesktopLabels:         desktop.GetAllLabels(),
	}
	s.emit(ctx, event)
}

func (s *WindowsService) emit(ctx context.Context, event events.AuditEvent) {
	if err := s.cfg.Emitter.EmitAuditEvent(ctx, event); err != nil {
		s.cfg.Log.WithError(err).Errorf("Failed to emit audit event %v", event)
	}
}

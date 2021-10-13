/*
Copyright 2018-2021 Gravitational, Inc.

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
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	mathrand "math/rand"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gravitational/teleport"
	apidefaults "github.com/gravitational/teleport/api/defaults"
	"github.com/gravitational/teleport/api/types"
	apievents "github.com/gravitational/teleport/api/types/events"
	apiutils "github.com/gravitational/teleport/api/utils"
	"github.com/gravitational/teleport/lib/auth"
	"github.com/gravitational/teleport/lib/defaults"
	"github.com/gravitational/teleport/lib/events"
	"github.com/gravitational/teleport/lib/events/filesessions"
	"github.com/gravitational/teleport/lib/httplib"
	kubeutils "github.com/gravitational/teleport/lib/kube/utils"
	"github.com/gravitational/teleport/lib/labels"
	"github.com/gravitational/teleport/lib/reversetunnel"
	"github.com/gravitational/teleport/lib/services"
	"github.com/gravitational/teleport/lib/session"
	"github.com/gravitational/teleport/lib/srv"
	"github.com/gravitational/teleport/lib/sshca"
	"github.com/gravitational/teleport/lib/utils"

	"github.com/gravitational/oxy/forward"
	fwdutils "github.com/gravitational/oxy/utils"
	"github.com/gravitational/trace"
	"github.com/gravitational/ttlmap"
	"github.com/jonboulle/clockwork"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
	utilexec "k8s.io/client-go/util/exec"
)

// KubeServiceType specifies a Teleport service type which can forward Kubernetes requests
type KubeServiceType int

const (
	// KubeService is a Teleport kubernetes_service. A KubeService always forwards
	// requests directly to a Kubernetes endpoint.
	KubeService KubeServiceType = iota
	// ProxyService is a Teleport proxy_service with kube_listen_addr/
	// kube_public_addr enabled. A ProxyService always forwards requests to a
	// Teleport KubeService or LegacyProxyService.
	ProxyService
	// LegacyProxyService is a Teleport proxy_service with the kubernetes section
	// enabled. A LegacyProxyService can forward requests directly to a Kubernetes
	// endpoint, or to another Teleport LegacyProxyService or KubeService.
	LegacyProxyService
)

// ForwarderConfig specifies configuration for proxy forwarder
type ForwarderConfig struct {
	// ReverseTunnelSrv is the teleport reverse tunnel server
	ReverseTunnelSrv reversetunnel.Server
	// ClusterName is a local cluster name
	ClusterName string
	// Keygen points to a key generator implementation
	Keygen sshca.Authority
	// Authz authenticates user
	Authz auth.Authorizer
	// AuthClient is a auth server client.
	AuthClient auth.ClientI
	// CachingAuthClient is a caching auth server client for read-only access.
	CachingAuthClient auth.AccessPoint
	// StreamEmitter is used to create audit streams
	// and emit audit events
	StreamEmitter events.StreamEmitter
	// DataDir is a data dir to store logs
	DataDir string
	// Namespace is a namespace of the proxy server (not a K8s namespace)
	Namespace string
	// ServerID is a unique ID of a proxy server
	ServerID string
	// ClusterOverride if set, routes all requests
	// to the cluster name, used in tests
	ClusterOverride string
	// Context passes the optional external context
	// passing global close to all forwarder operations
	Context context.Context
	// KubeconfigPath is a path to kubernetes configuration
	KubeconfigPath string
	// KubeServiceType specifies which Teleport service type this forwarder is for
	KubeServiceType KubeServiceType
	// KubeClusterName is the name of the kubernetes cluster that this
	// forwarder handles.
	KubeClusterName string
	// Clock is a server clock, could be overridden in tests
	Clock clockwork.Clock
	// ConnPingPeriod is a period for sending ping messages on the incoming
	// connection.
	ConnPingPeriod time.Duration
	// Component name to include in log output.
	Component string
	// StaticLabels is map of static labels associated with this cluster.
	// Used for RBAC.
	StaticLabels map[string]string
	// DynamicLabels is map of dynamic labels associated with this cluster.
	// Used for RBAC.
	DynamicLabels *labels.Dynamic
	// LockWatcher is a lock watcher.
	LockWatcher *services.LockWatcher
	// CheckImpersonationPermissions is an optional override of the default
	// impersonation permissions check, for use in testing
	CheckImpersonationPermissions ImpersonationPermissionsChecker
	// PublicAddr is the address that can be used to reach the kube cluster
	PublicAddr string
}

// CheckAndSetDefaults checks and sets default values
func (f *ForwarderConfig) CheckAndSetDefaults() error {
	if f.AuthClient == nil {
		return trace.BadParameter("missing parameter AuthClient")
	}
	if f.CachingAuthClient == nil {
		return trace.BadParameter("missing parameter CachingAuthClient")
	}
	if f.Authz == nil {
		return trace.BadParameter("missing parameter Authz")
	}
	if f.StreamEmitter == nil {
		return trace.BadParameter("missing parameter StreamEmitter")
	}
	if f.ClusterName == "" {
		return trace.BadParameter("missing parameter ClusterName")
	}
	if f.Keygen == nil {
		return trace.BadParameter("missing parameter Keygen")
	}
	if f.DataDir == "" {
		return trace.BadParameter("missing parameter DataDir")
	}
	if f.ServerID == "" {
		return trace.BadParameter("missing parameter ServerID")
	}
	if f.Namespace == "" {
		f.Namespace = apidefaults.Namespace
	}
	if f.Context == nil {
		f.Context = context.TODO()
	}
	if f.Clock == nil {
		f.Clock = clockwork.NewRealClock()
	}
	if f.ConnPingPeriod == 0 {
		f.ConnPingPeriod = defaults.HighResPollingPeriod
	}
	if f.Component == "" {
		f.Component = "kube_forwarder"
	}
	switch f.KubeServiceType {
	case KubeService:
	case ProxyService:
	case LegacyProxyService:
	default:
		return trace.BadParameter("unknown value for KubeServiceType")
	}
	if f.KubeClusterName == "" && f.KubeconfigPath == "" && f.KubeServiceType == LegacyProxyService {
		// Running without a kubeconfig and explicit k8s cluster name. Use
		// teleport cluster name instead, to ask kubeutils.GetKubeConfig to
		// attempt loading the in-cluster credentials.
		f.KubeClusterName = f.ClusterName
	}
	return nil
}

// NewForwarder returns new instance of Kubernetes request
// forwarding proxy.
func NewForwarder(cfg ForwarderConfig) (*Forwarder, error) {
	if err := cfg.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}
	log := log.WithFields(log.Fields{
		trace.Component: cfg.Component,
	})

	// Pick the permissions check function to use, applying an override
	// if specified.
	checkImpersonation := checkImpersonationPermissions
	if cfg.CheckImpersonationPermissions != nil {
		checkImpersonation = cfg.CheckImpersonationPermissions
	}

	creds, err := getKubeCreds(cfg.Context, log, cfg.ClusterName, cfg.KubeClusterName, cfg.KubeconfigPath, cfg.KubeServiceType, checkImpersonation)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	clientCredentials, err := ttlmap.New(defaults.ClientCacheSize)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	closeCtx, close := context.WithCancel(cfg.Context)
	fwd := &Forwarder{
		creds:             creds,
		log:               log,
		router:            *httprouter.New(),
		cfg:               cfg,
		clientCredentials: clientCredentials,
		activeRequests:    make(map[string]context.Context),
		ctx:               closeCtx,
		close:             close,
	}

	fwd.router.POST("/api/:ver/namespaces/:podNamespace/pods/:podName/exec", fwd.withAuth(fwd.exec))
	fwd.router.GET("/api/:ver/namespaces/:podNamespace/pods/:podName/exec", fwd.withAuth(fwd.exec))

	fwd.router.POST("/api/:ver/namespaces/:podNamespace/pods/:podName/attach", fwd.withAuth(fwd.exec))
	fwd.router.GET("/api/:ver/namespaces/:podNamespace/pods/:podName/attach", fwd.withAuth(fwd.exec))

	fwd.router.POST("/api/:ver/namespaces/:podNamespace/pods/:podName/portforward", fwd.withAuth(fwd.portForward))
	fwd.router.GET("/api/:ver/namespaces/:podNamespace/pods/:podName/portforward", fwd.withAuth(fwd.portForward))

	fwd.router.NotFound = fwd.withAuthStd(fwd.catchAll)

	if cfg.ClusterOverride != "" {
		fwd.log.Debugf("Cluster override is set, forwarder will send all requests to remote cluster %v.", cfg.ClusterOverride)
	}
	return fwd, nil
}

// Forwarder intercepts kubernetes requests, acting as Kubernetes API proxy.
// it blindly forwards most of the requests on HTTPS protocol layer,
// however some requests like exec sessions it intercepts and records.
type Forwarder struct {
	mu     sync.Mutex
	log    log.FieldLogger
	router httprouter.Router
	cfg    ForwarderConfig
	// clientCredentials is an expiring cache of ephemeral client credentials.
	// Forwarder requests credentials with client identity, when forwarding to
	// another teleport process (but not when forwarding to k8s API).
	//
	// TODO(klizhentas): flush certs on teleport CA rotation?
	clientCredentials *ttlmap.TTLMap
	// activeRequests is a map used to serialize active CSR requests to the auth server
	activeRequests map[string]context.Context
	// close is a close function
	close context.CancelFunc
	// ctx is a global context signalling exit
	ctx context.Context
	// creds contain kubernetes credentials for multiple clusters.
	// map key is cluster name.
	creds map[string]*kubeCreds
}

// Close signals close to all outstanding or background operations
// to complete
func (f *Forwarder) Close() error {
	f.close()
	return nil
}

func (f *Forwarder) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	f.router.ServeHTTP(rw, r)
}

// authContext is a context of authenticated user,
// contains information about user, target cluster and authenticated groups
type authContext struct {
	auth.Context
	kubeGroups               map[string]struct{}
	kubeUsers                map[string]struct{}
	kubeCluster              string
	teleportCluster          teleportClusterClient
	teleportClusterEndpoints []endpoint
	recordingConfig          types.SessionRecordingConfig
	// clientIdleTimeout sets information on client idle timeout
	clientIdleTimeout time.Duration
	// disconnectExpiredCert if set, controls the time when the connection
	// should be disconnected because the client cert expires
	disconnectExpiredCert time.Time
	// sessionTTL specifies the duration of the user's session
	sessionTTL time.Duration
}

type endpoint struct {
	// addr is a direct network address.
	addr string
	// serverID is the server:cluster ID of the endpoint,
	// which is used to find its corresponding reverse tunnel.
	serverID string
}

func (c authContext) String() string {
	return fmt.Sprintf("user: %v, users: %v, groups: %v, teleport cluster: %v, kube cluster: %v", c.User.GetName(), c.kubeUsers, c.kubeGroups, c.teleportCluster.name, c.kubeCluster)
}

func (c *authContext) key() string {
	// it is important that the context key contains user, kubernetes groups and certificate expiry,
	// so that new logins with different parameters will not reuse this context
	return fmt.Sprintf("%v:%v:%v:%v:%v:%v", c.teleportCluster.name, c.User.GetName(), c.kubeUsers, c.kubeGroups, c.kubeCluster, c.disconnectExpiredCert.UTC().Unix())
}

func (c *authContext) eventClusterMeta() apievents.KubernetesClusterMetadata {
	return apievents.KubernetesClusterMetadata{
		KubernetesCluster: c.kubeCluster,
		KubernetesUsers:   utils.StringsSliceFromSet(c.kubeUsers),
		KubernetesGroups:  utils.StringsSliceFromSet(c.kubeGroups),
	}
}

type dialFunc func(ctx context.Context, network, addr, serverID string) (net.Conn, error)

// teleportClusterClient is a client for either a k8s endpoint in local cluster or a
// proxy endpoint in a remote cluster.
type teleportClusterClient struct {
	remoteAddr utils.NetAddr
	name       string
	dial       dialFunc
	// targetAddr is a direct network address.
	targetAddr string
	// serverID is the server:cluster ID of the endpoint,
	// which is used to find its corresponding reverse tunnel.
	serverID       string
	isRemote       bool
	isRemoteClosed func() bool
}

func (c *teleportClusterClient) DialWithContext(ctx context.Context, network, _ string) (net.Conn, error) {
	return c.dial(ctx, network, c.targetAddr, c.serverID)
}

// handlerWithAuthFunc is http handler with passed auth context
type handlerWithAuthFunc func(ctx *authContext, w http.ResponseWriter, r *http.Request, p httprouter.Params) (interface{}, error)

// handlerWithAuthFuncStd is http handler with passed auth context
type handlerWithAuthFuncStd func(ctx *authContext, w http.ResponseWriter, r *http.Request) (interface{}, error)

// authenticate function authenticates request
func (f *Forwarder) authenticate(req *http.Request) (*authContext, error) {
	const accessDeniedMsg = "[00] access denied"

	var isRemoteUser bool
	userTypeI := req.Context().Value(auth.ContextUser)
	switch userTypeI.(type) {
	case auth.LocalUser:

	case auth.RemoteUser:
		isRemoteUser = true
	case auth.BuiltinRole:
		f.log.Warningf("Denying proxy access to unauthenticated user of type %T - this can sometimes be caused by inadvertently using an HTTP load balancer instead of a TCP load balancer on the Kubernetes port.", userTypeI)
		return nil, trace.AccessDenied(accessDeniedMsg)
	default:
		f.log.Warningf("Denying proxy access to unsupported user type: %T.", userTypeI)
		return nil, trace.AccessDenied(accessDeniedMsg)
	}

	userContext, err := f.cfg.Authz.Authorize(req.Context())
	if err != nil {
		switch {
		// propagate connection problem error so we can differentiate
		// between connection failed and access denied
		case trace.IsConnectionProblem(err):
			return nil, trace.ConnectionProblem(err, "[07] failed to connect to the database")
		case trace.IsAccessDenied(err):
			// don't print stack trace, just log the warning
			f.log.Warn(err)
			return nil, trace.AccessDenied(accessDeniedMsg)
		default:
			f.log.Warn(trace.DebugReport(err))
			return nil, trace.AccessDenied(accessDeniedMsg)
		}
	}
	peers := req.TLS.PeerCertificates
	if len(peers) > 1 {
		// when turning intermediaries on, don't forget to verify
		// https://github.com/kubernetes/kubernetes/pull/34524/files#diff-2b283dde198c92424df5355f39544aa4R59
		return nil, trace.AccessDenied("access denied: intermediaries are not supported")
	}
	if len(peers) == 0 {
		return nil, trace.AccessDenied("access denied: only mutual TLS authentication is supported")
	}
	clientCert := peers[0]
	authContext, err := f.setupContext(*userContext, req, isRemoteUser, clientCert.NotAfter)
	if err != nil {
		f.log.Warn(err.Error())
		if trace.IsAccessDenied(err) {
			return nil, trace.AccessDenied(accessDeniedMsg)
		}
		return nil, trace.Wrap(err)
	}
	return authContext, nil
}

func (f *Forwarder) withAuthStd(handler handlerWithAuthFuncStd) http.HandlerFunc {
	return httplib.MakeStdHandlerWithErrorWriter(func(w http.ResponseWriter, req *http.Request) (interface{}, error) {
		authContext, err := f.authenticate(req)
		if err != nil {
			return nil, trace.Wrap(err)
		}
		if err := f.authorize(req.Context(), authContext); err != nil {
			return nil, trace.Wrap(err)
		}

		return handler(authContext, w, req)
	}, f.formatResponseError)
}

func (f *Forwarder) withAuth(handler handlerWithAuthFunc) httprouter.Handle {
	return httplib.MakeHandlerWithErrorWriter(func(w http.ResponseWriter, req *http.Request, p httprouter.Params) (interface{}, error) {
		authContext, err := f.authenticate(req)
		if err != nil {
			return nil, trace.Wrap(err)
		}
		if err := f.authorize(req.Context(), authContext); err != nil {
			return nil, trace.Wrap(err)
		}
		return handler(authContext, w, req, p)
	}, f.formatResponseError)
}

func (f *Forwarder) formatForwardResponseError(rw http.ResponseWriter, r *http.Request, respErr error) {
	f.formatResponseError(rw, respErr)
}

func (f *Forwarder) formatResponseError(rw http.ResponseWriter, respErr error) {
	status := &metav1.Status{
		Status: metav1.StatusFailure,
		// Don't trace.Unwrap the error, in case it was wrapped with a
		// user-friendly message. The underlying root error is likely too
		// low-level to be useful.
		Message: respErr.Error(),
		Code:    int32(trace.ErrorToCode(respErr)),
	}
	data, err := runtime.Encode(statusCodecs.LegacyCodec(), status)
	if err != nil {
		f.log.Warningf("Failed encoding error into kube Status object: %v", err)
		trace.WriteError(rw, respErr)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	// Always write InternalServerError, that's the only code that kubectl will
	// parse the Status object for. The Status object has the real status code
	// embedded.
	rw.WriteHeader(http.StatusInternalServerError)
	if _, err := rw.Write(data); err != nil {
		f.log.Warningf("Failed writing kube error response body: %v", err)
	}
}

func (f *Forwarder) setupContext(ctx auth.Context, req *http.Request, isRemoteUser bool, certExpires time.Time) (*authContext, error) {
	roles := ctx.Checker

	// adjust session ttl to the smaller of two values: the session
	// ttl requested in tsh or the session ttl for the role.
	sessionTTL := roles.AdjustSessionTTL(time.Hour)

	identity := ctx.Identity.GetIdentity()
	teleportClusterName := identity.RouteToCluster
	if teleportClusterName == "" {
		teleportClusterName = f.cfg.ClusterName
	}
	isRemoteCluster := f.cfg.ClusterName != teleportClusterName

	if isRemoteCluster && isRemoteUser {
		return nil, trace.AccessDenied("access denied: remote user can not access remote cluster")
	}

	var kubeUsers, kubeGroups []string
	// Only check k8s principals for local clusters.
	//
	// For remote clusters, everything will be remapped to new roles on the
	// leaf and checked there.
	if !isRemoteCluster {
		var err error
		// check signing TTL and return a list of allowed logins
		kubeGroups, kubeUsers, err = roles.CheckKubeGroupsAndUsers(sessionTTL, false)
		if err != nil {
			return nil, trace.Wrap(err)
		}
	}

	// By default, if no kubernetes_users is set (which will be a majority),
	// user will impersonate themselves, which is the backwards-compatible behavior.
	if len(kubeUsers) == 0 {
		kubeUsers = append(kubeUsers, ctx.User.GetName())
	}

	// KubeSystemAuthenticated is a builtin group that allows
	// any user to access common API methods, e.g. discovery methods
	// required for initial client usage, without it, restricted user's
	// kubectl clients will not work
	if !apiutils.SliceContainsStr(kubeGroups, teleport.KubeSystemAuthenticated) {
		kubeGroups = append(kubeGroups, teleport.KubeSystemAuthenticated)
	}

	// Get a dialer for either a k8s endpoint in current cluster or a tunneled
	// endpoint for a leaf teleport cluster.
	var dialFn dialFunc
	var isRemoteClosed func() bool
	if isRemoteCluster {
		// Tunnel is nil for a teleport process with "kubernetes_service" but
		// not "proxy_service".
		if f.cfg.ReverseTunnelSrv == nil {
			return nil, trace.BadParameter("this Teleport process can not dial Kubernetes endpoints in remote Teleport clusters; only proxy_service supports this, make sure a Teleport proxy is first in the request path")
		}

		targetCluster, err := f.cfg.ReverseTunnelSrv.GetSite(teleportClusterName)
		if err != nil {
			return nil, trace.Wrap(err)
		}

		dialFn = func(ctx context.Context, network, addr, serverID string) (net.Conn, error) {
			return targetCluster.DialTCP(reversetunnel.DialParams{
				From:     &utils.NetAddr{AddrNetwork: "tcp", Addr: req.RemoteAddr},
				To:       &utils.NetAddr{AddrNetwork: "tcp", Addr: addr},
				ConnType: types.KubeTunnel,
				ServerID: serverID,
			})
		}
		isRemoteClosed = targetCluster.IsClosed
	} else if f.cfg.ReverseTunnelSrv != nil {
		// Not a remote cluster and we have a reverse tunnel server.
		// Use the local reversetunnel.Site which knows how to dial by serverID
		// (for "kubernetes_service" connected over a tunnel) and falls back to
		// direct dial if needed.
		localCluster, err := f.cfg.ReverseTunnelSrv.GetSite(f.cfg.ClusterName)
		if err != nil {
			return nil, trace.Wrap(err)
		}

		dialFn = func(ctx context.Context, network, addr, serverID string) (net.Conn, error) {
			return localCluster.DialTCP(reversetunnel.DialParams{
				From:     &utils.NetAddr{AddrNetwork: "tcp", Addr: req.RemoteAddr},
				To:       &utils.NetAddr{AddrNetwork: "tcp", Addr: addr},
				ConnType: types.KubeTunnel,
				ServerID: serverID,
			})
		}
		isRemoteClosed = localCluster.IsClosed
	} else {
		// Don't have a reverse tunnel server, so we can only dial directly.
		dialFn = func(ctx context.Context, network, addr, _ string) (net.Conn, error) {
			return new(net.Dialer).DialContext(ctx, network, addr)
		}
		isRemoteClosed = func() bool { return false }
	}

	netConfig, err := f.cfg.CachingAuthClient.GetClusterNetworkingConfig(f.ctx)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	recordingConfig, err := f.cfg.CachingAuthClient.GetSessionRecordingConfig(f.ctx)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	authCtx := &authContext{
		clientIdleTimeout: roles.AdjustClientIdleTimeout(netConfig.GetClientIdleTimeout()),
		sessionTTL:        sessionTTL,
		Context:           ctx,
		kubeGroups:        utils.StringsSet(kubeGroups),
		kubeUsers:         utils.StringsSet(kubeUsers),
		recordingConfig:   recordingConfig,
		teleportCluster: teleportClusterClient{
			name:           teleportClusterName,
			remoteAddr:     utils.NetAddr{AddrNetwork: "tcp", Addr: req.RemoteAddr},
			dial:           dialFn,
			isRemote:       isRemoteCluster,
			isRemoteClosed: isRemoteClosed,
		},
	}

	authCtx.kubeCluster = identity.KubernetesCluster
	if !isRemoteCluster {
		kubeCluster, err := kubeutils.CheckOrSetKubeCluster(req.Context(), f.cfg.CachingAuthClient, identity.KubernetesCluster, teleportClusterName)
		if err != nil {
			if !trace.IsNotFound(err) {
				return nil, trace.Wrap(err)
			}
			// Fallback for old clusters and old user certs. Assume that the
			// user is trying to access the default cluster name.
			kubeCluster = teleportClusterName
		}
		authCtx.kubeCluster = kubeCluster
	}

	authPref, err := f.cfg.CachingAuthClient.GetAuthPreference(req.Context())
	if err != nil {
		return nil, trace.Wrap(err)
	}

	disconnectExpiredCert := roles.AdjustDisconnectExpiredCert(authPref.GetDisconnectExpiredCert())
	if !certExpires.IsZero() && disconnectExpiredCert {
		authCtx.disconnectExpiredCert = certExpires
	}

	return authCtx, nil
}

func (f *Forwarder) authorize(ctx context.Context, actx *authContext) error {
	if actx.teleportCluster.isRemote {
		// Authorization for a remote kube cluster will happen on the remote
		// end (by their proxy), after that cluster has remapped used roles.
		f.log.WithField("auth_context", actx.String()).Debug("Skipping authorization for a remote kubernetes cluster name")
		return nil
	}
	if actx.kubeCluster == "" {
		// This should only happen for remote clusters (filtered above), but
		// check and report anyway.
		f.log.WithField("auth_context", actx.String()).Debug("Skipping authorization due to unknown kubernetes cluster name")
		return nil
	}
	servers, err := f.cfg.CachingAuthClient.GetKubeServices(ctx)
	if err != nil {
		return trace.Wrap(err)
	}
	ap, err := f.cfg.CachingAuthClient.GetAuthPreference(ctx)
	if err != nil {
		return trace.Wrap(err)
	}
	mfaParams := services.AccessMFAParams{
		Verified:       actx.Identity.GetIdentity().MFAVerified != "",
		AlwaysRequired: ap.GetRequireSessionMFA(),
	}
	// Check authz against the first match.
	//
	// We assume that users won't register two identically-named clusters with
	// mis-matched labels. If they do, expect weirdness.
	clusterNotFound := trace.AccessDenied("kubernetes cluster %q not found", actx.kubeCluster)
	for _, s := range servers {
		for _, ks := range s.GetKubernetesClusters() {
			if ks.Name != actx.kubeCluster {
				continue
			}
			kV3, err := types.NewKubernetesClusterV3FromLegacyCluster(s.GetNamespace(), ks)
			if err != nil {
				return trace.Wrap(err)
			}
			if err := actx.Checker.CheckAccess(kV3, mfaParams); err != nil {
				return clusterNotFound
			}
			return nil
		}
	}
	if actx.kubeCluster == f.cfg.ClusterName {
		f.log.WithField("auth_context", actx.String()).Debug("Skipping authorization for proxy-based kubernetes cluster,")
		return nil
	}
	return clusterNotFound
}

// newStreamer returns sync or async streamer based on the configuration
// of the server and the session, sync streamer sends the events
// directly to the auth server and blocks if the events can not be received,
// async streamer buffers the events to disk and uploads the events later
func (f *Forwarder) newStreamer(ctx *authContext) (events.Streamer, error) {
	if services.IsRecordSync(ctx.recordingConfig.GetMode()) {
		f.log.Debugf("Using sync streamer for session.")
		return f.cfg.AuthClient, nil
	}
	f.log.Debugf("Using async streamer for session.")
	dir := filepath.Join(
		f.cfg.DataDir, teleport.LogsDir, teleport.ComponentUpload,
		events.StreamingLogsDir, apidefaults.Namespace,
	)
	fileStreamer, err := filesessions.NewStreamer(dir)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	// TeeStreamer sends non-print and non disk events
	// to the audit log in async mode, while buffering all
	// events on disk for further upload at the end of the session
	return events.NewTeeStreamer(fileStreamer, f.cfg.StreamEmitter), nil
}

// exec forwards all exec requests to the target server, captures
// all output from the session
func (f *Forwarder) exec(ctx *authContext, w http.ResponseWriter, req *http.Request, p httprouter.Params) (resp interface{}, err error) {
	f.log.Debugf("Exec %v.", req.URL.String())
	defer func() {
		if err != nil {
			f.log.WithError(err).Debug("Exec request failed")
		}
	}()

	sess, err := f.newClusterSession(*ctx)
	if err != nil {
		// This error goes to kubernetes client and is not visible in the logs
		// of the teleport server if not logged here.
		f.log.Errorf("Failed to create cluster session: %v.", err)
		return nil, trace.Wrap(err)
	}
	sessionStart := f.cfg.Clock.Now().UTC()

	q := req.URL.Query()
	request := remoteCommandRequest{
		podNamespace:       p.ByName("podNamespace"),
		podName:            p.ByName("podName"),
		containerName:      q.Get("container"),
		cmd:                q["command"],
		stdin:              utils.AsBool(q.Get("stdin")),
		stdout:             utils.AsBool(q.Get("stdout")),
		stderr:             utils.AsBool(q.Get("stderr")),
		tty:                utils.AsBool(q.Get("tty")),
		httpRequest:        req,
		httpResponseWriter: w,
		context:            req.Context(),
		pingPeriod:         f.cfg.ConnPingPeriod,
	}
	eventPodMeta := request.eventPodMeta(request.context, sess.creds)

	var recorder events.SessionRecorder
	var emitter apievents.Emitter
	sessionID := session.NewID()
	if sess.noAuditEvents {
		// All events should be recorded by kubernetes_service and not proxy_service
		emitter = events.NewDiscardEmitter()
		request.onResize = func(resize remotecommand.TerminalSize) {}
	} else if request.tty {
		streamer, err := f.newStreamer(ctx)
		if err != nil {
			return nil, trace.Wrap(err)
		}
		// create session recorder
		// get the audit log from the server and create a session recorder. this will
		// be a discard audit log if the proxy is in recording mode and a teleport
		// node so we don't create double recordings.
		recorder, err = events.NewAuditWriter(events.AuditWriterConfig{
			// Audit stream is using server context, not session context,
			// to make sure that session is uploaded even after it is closed
			Context:      f.ctx,
			Streamer:     streamer,
			Clock:        f.cfg.Clock,
			SessionID:    sessionID,
			ServerID:     f.cfg.ServerID,
			Namespace:    f.cfg.Namespace,
			RecordOutput: ctx.recordingConfig.GetMode() != types.RecordOff,
			Component:    teleport.Component(teleport.ComponentSession, teleport.ComponentProxyKube),
			ClusterName:  f.cfg.ClusterName,
		})
		if err != nil {
			return nil, trace.Wrap(err)
		}
		emitter = recorder
		defer recorder.Close(f.ctx)
		request.onResize = func(resize remotecommand.TerminalSize) {
			params := session.TerminalParams{
				W: int(resize.Width),
				H: int(resize.Height),
			}
			// Build the resize event.
			resizeEvent := &apievents.Resize{
				Metadata: apievents.Metadata{
					Type:        events.ResizeEvent,
					Code:        events.TerminalResizeCode,
					ClusterName: f.cfg.ClusterName,
				},
				ConnectionMetadata: apievents.ConnectionMetadata{
					RemoteAddr: req.RemoteAddr,
					Protocol:   events.EventProtocolKube,
				},
				ServerMetadata: apievents.ServerMetadata{
					ServerNamespace: f.cfg.Namespace,
				},
				SessionMetadata: apievents.SessionMetadata{
					SessionID: string(sessionID),
					WithMFA:   ctx.Identity.GetIdentity().MFAVerified,
				},
				UserMetadata: apievents.UserMetadata{
					User:         ctx.User.GetName(),
					Login:        ctx.User.GetName(),
					Impersonator: ctx.Identity.GetIdentity().Impersonator,
				},
				TerminalSize:              params.Serialize(),
				KubernetesClusterMetadata: ctx.eventClusterMeta(),
				KubernetesPodMetadata:     eventPodMeta,
			}

			// Report the updated window size to the event log (this is so the sessions
			// can be replayed correctly).
			if err := recorder.EmitAuditEvent(f.ctx, resizeEvent); err != nil {
				f.log.WithError(err).Warn("Failed to emit terminal resize event.")
			}
		}
	} else {
		emitter = f.cfg.StreamEmitter
	}

	if request.tty {
		// Emit "new session created" event. There are no initial terminal
		// parameters per k8s protocol, so set up with any default
		termParams := session.TerminalParams{
			W: 100,
			H: 100,
		}
		sessionStartEvent := &apievents.SessionStart{
			Metadata: apievents.Metadata{
				Type:        events.SessionStartEvent,
				Code:        events.SessionStartCode,
				ClusterName: f.cfg.ClusterName,
			},
			ServerMetadata: apievents.ServerMetadata{
				ServerID:        f.cfg.ServerID,
				ServerNamespace: f.cfg.Namespace,
				ServerHostname:  sess.teleportCluster.name,
				ServerAddr:      sess.teleportCluster.targetAddr,
			},
			SessionMetadata: apievents.SessionMetadata{
				SessionID: string(sessionID),
				WithMFA:   ctx.Identity.GetIdentity().MFAVerified,
			},
			UserMetadata: apievents.UserMetadata{
				User:         ctx.User.GetName(),
				Login:        ctx.User.GetName(),
				Impersonator: ctx.Identity.GetIdentity().Impersonator,
			},
			ConnectionMetadata: apievents.ConnectionMetadata{
				RemoteAddr: req.RemoteAddr,
				LocalAddr:  sess.teleportCluster.targetAddr,
				Protocol:   events.EventProtocolKube,
			},
			TerminalSize:              termParams.Serialize(),
			KubernetesClusterMetadata: ctx.eventClusterMeta(),
			KubernetesPodMetadata:     eventPodMeta,
			InitialCommand:            request.cmd,
			SessionRecording:          ctx.recordingConfig.GetMode(),
		}
		if err := emitter.EmitAuditEvent(f.ctx, sessionStartEvent); err != nil {
			f.log.WithError(err).Warn("Failed to emit event.")
		}
	}

	if err := f.setupForwardingHeaders(sess, req); err != nil {
		return nil, trace.Wrap(err)
	}

	proxy, err := createRemoteCommandProxy(request)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer proxy.Close()

	executor, err := f.getExecutor(*ctx, sess, req)
	if err != nil {
		f.log.WithError(err).Warning("Failed creating executor.")
		return nil, trace.Wrap(err)
	}
	streamOptions := proxy.options()

	// Wrap stdin/out/err with data trackers, but keep them as nil if they were
	// nil. Otherwise, executor will try to use these tracking read/writers
	// when the underlying stream is nil.
	trackIn := utils.NewTrackingReader(streamOptions.Stdin)
	if streamOptions.Stdin != nil {
		streamOptions.Stdin = trackIn
	}
	trackOut := utils.NewTrackingWriter(streamOptions.Stdout)
	if streamOptions.Stdout != nil {
		streamOptions.Stdout = trackOut
	}
	trackErr := utils.NewTrackingWriter(streamOptions.Stderr)
	if streamOptions.Stderr != nil {
		streamOptions.Stderr = trackErr
	}
	if recorder != nil {
		// capture stderr and stdout writes to session recorder
		streamOptions.Stdout = utils.NewBroadcastWriter(streamOptions.Stdout, recorder)
		streamOptions.Stderr = utils.NewBroadcastWriter(streamOptions.Stderr, recorder)
	}

	// Defer a cleanup handler that will mark the stream as complete on exit, regardless of
	// whether it exits successfully, or with an error.
	// NOTE that this cleanup handler MAY MODIFY the returned error value.
	defer func() {
		if err := proxy.sendStatus(err); err != nil {
			f.log.WithError(err).Warning("Failed to send status. Exec command was aborted by client.")
		}

		if request.tty {
			sessionDataEvent := &apievents.SessionData{
				Metadata: apievents.Metadata{
					Type:        events.SessionDataEvent,
					Code:        events.SessionDataCode,
					ClusterName: f.cfg.ClusterName,
				},
				ServerMetadata: apievents.ServerMetadata{
					ServerID:        f.cfg.ServerID,
					ServerNamespace: f.cfg.Namespace,
				},
				SessionMetadata: apievents.SessionMetadata{
					SessionID: string(sessionID),
					WithMFA:   ctx.Identity.GetIdentity().MFAVerified,
				},
				UserMetadata: apievents.UserMetadata{
					User:         ctx.User.GetName(),
					Login:        ctx.User.GetName(),
					Impersonator: ctx.Identity.GetIdentity().Impersonator,
				},
				ConnectionMetadata: apievents.ConnectionMetadata{
					RemoteAddr: req.RemoteAddr,
					LocalAddr:  sess.teleportCluster.targetAddr,
					Protocol:   events.EventProtocolKube,
				},
				// Bytes transmitted from user to pod.
				BytesTransmitted: trackIn.Count(),
				// Bytes received from pod by user.
				BytesReceived: trackOut.Count() + trackErr.Count(),
			}
			if err := emitter.EmitAuditEvent(f.ctx, sessionDataEvent); err != nil {
				f.log.WithError(err).Warn("Failed to emit session data event.")
			}
			sessionEndEvent := &apievents.SessionEnd{
				Metadata: apievents.Metadata{
					Type:        events.SessionEndEvent,
					Code:        events.SessionEndCode,
					ClusterName: f.cfg.ClusterName,
				},
				ServerMetadata: apievents.ServerMetadata{
					ServerID:        f.cfg.ServerID,
					ServerNamespace: f.cfg.Namespace,
				},
				SessionMetadata: apievents.SessionMetadata{
					SessionID: string(sessionID),
					WithMFA:   ctx.Identity.GetIdentity().MFAVerified,
				},
				UserMetadata: apievents.UserMetadata{
					User:         ctx.User.GetName(),
					Login:        ctx.User.GetName(),
					Impersonator: ctx.Identity.GetIdentity().Impersonator,
				},
				ConnectionMetadata: apievents.ConnectionMetadata{
					RemoteAddr: req.RemoteAddr,
					LocalAddr:  sess.teleportCluster.targetAddr,
					Protocol:   events.EventProtocolKube,
				},
				Interactive: true,
				// There can only be 1 participant, k8s sessions are not join-able.
				Participants:              []string{ctx.User.GetName()},
				StartTime:                 sessionStart,
				EndTime:                   f.cfg.Clock.Now().UTC(),
				KubernetesClusterMetadata: ctx.eventClusterMeta(),
				KubernetesPodMetadata:     eventPodMeta,
				InitialCommand:            request.cmd,
				SessionRecording:          ctx.recordingConfig.GetMode(),
			}
			if err := emitter.EmitAuditEvent(f.ctx, sessionEndEvent); err != nil {
				f.log.WithError(err).Warn("Failed to emit session end event.")
			}
		} else {
			// send an exec event
			execEvent := &apievents.Exec{
				Metadata: apievents.Metadata{
					Type:        events.ExecEvent,
					ClusterName: f.cfg.ClusterName,
				},
				ServerMetadata: apievents.ServerMetadata{
					ServerID:        f.cfg.ServerID,
					ServerNamespace: f.cfg.Namespace,
				},
				SessionMetadata: apievents.SessionMetadata{
					SessionID: string(sessionID),
					WithMFA:   ctx.Identity.GetIdentity().MFAVerified,
				},
				UserMetadata: apievents.UserMetadata{
					User:         ctx.User.GetName(),
					Login:        ctx.User.GetName(),
					Impersonator: ctx.Identity.GetIdentity().Impersonator,
				},
				ConnectionMetadata: apievents.ConnectionMetadata{
					RemoteAddr: req.RemoteAddr,
					LocalAddr:  sess.teleportCluster.targetAddr,
					Protocol:   events.EventProtocolKube,
				},
				CommandMetadata: apievents.CommandMetadata{
					Command: strings.Join(request.cmd, " "),
				},
				KubernetesClusterMetadata: ctx.eventClusterMeta(),
				KubernetesPodMetadata:     eventPodMeta,
			}
			if err != nil {
				execEvent.Code = events.ExecFailureCode
				execEvent.Error = err.Error()
				if exitErr, ok := err.(utilexec.ExitError); ok && exitErr.Exited() {
					execEvent.ExitCode = fmt.Sprintf("%d", exitErr.ExitStatus())
				}
			} else {
				execEvent.Code = events.ExecCode
			}
			if err := emitter.EmitAuditEvent(f.ctx, execEvent); err != nil {
				f.log.WithError(err).Warn("Failed to emit event.")
			}
		}
	}()

	if err = executor.Stream(streamOptions); err != nil {
		f.log.WithError(err).Warning("Executor failed while streaming.")
		return nil, trace.Wrap(err)
	}

	return nil, nil
}

// portForward starts port forwarding to the remote cluster
func (f *Forwarder) portForward(ctx *authContext, w http.ResponseWriter, req *http.Request, p httprouter.Params) (interface{}, error) {
	f.log.Debugf("Port forward: %v. req headers: %v.", req.URL.String(), req.Header)
	sess, err := f.newClusterSession(*ctx)
	if err != nil {
		// This error goes to kubernetes client and is not visible in the logs
		// of the teleport server if not logged here.
		f.log.Errorf("Failed to create cluster session: %v.", err)
		return nil, trace.Wrap(err)
	}

	if err := f.setupForwardingHeaders(sess, req); err != nil {
		f.log.Debugf("DENIED Port forward: %v.", req.URL.String())
		return nil, trace.Wrap(err)
	}

	dialer, err := f.getDialer(*ctx, sess, req)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	onPortForward := func(addr string, success bool) {
		if sess.noAuditEvents {
			return
		}
		portForward := &apievents.PortForward{
			Metadata: apievents.Metadata{
				Type: events.PortForwardEvent,
				Code: events.PortForwardCode,
			},
			UserMetadata: apievents.UserMetadata{
				Login:        ctx.User.GetName(),
				User:         ctx.User.GetName(),
				Impersonator: ctx.Identity.GetIdentity().Impersonator,
			},
			ConnectionMetadata: apievents.ConnectionMetadata{
				LocalAddr:  sess.teleportCluster.targetAddr,
				RemoteAddr: req.RemoteAddr,
				Protocol:   events.EventProtocolKube,
			},
			Addr: addr,
			Status: apievents.Status{
				Success: success,
			},
		}
		if !success {
			portForward.Code = events.PortForwardFailureCode
		}
		if err := f.cfg.StreamEmitter.EmitAuditEvent(f.ctx, portForward); err != nil {
			f.log.WithError(err).Warn("Failed to emit event.")
		}
	}

	q := req.URL.Query()
	request := portForwardRequest{
		podNamespace:       p.ByName("podNamespace"),
		podName:            p.ByName("podName"),
		ports:              q["ports"],
		context:            req.Context(),
		httpRequest:        req,
		httpResponseWriter: w,
		onPortForward:      onPortForward,
		targetDialer:       dialer,
		pingPeriod:         f.cfg.ConnPingPeriod,
	}
	f.log.Debugf("Starting %v.", request)
	err = runPortForwarding(request)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	f.log.Debugf("Done %v.", request)
	return nil, nil
}

const (
	// ImpersonateHeaderPrefix is K8s impersonation prefix for impersonation feature:
	// https://kubernetes.io/docs/reference/access-authn-authz/authentication/#user-impersonation
	ImpersonateHeaderPrefix = "Impersonate-"
	// ImpersonateUserHeader is impersonation header for users
	ImpersonateUserHeader = "Impersonate-User"
	// ImpersonateGroupHeader is K8s impersonation header for user
	ImpersonateGroupHeader = "Impersonate-Group"
	// ImpersonationRequestDeniedMessage is access denied message for impersonation
	ImpersonationRequestDeniedMessage = "impersonation request has been denied"
)

func (f *Forwarder) setupForwardingHeaders(sess *clusterSession, req *http.Request) error {
	if err := setupImpersonationHeaders(f.log, sess.authContext, req.Header); err != nil {
		return trace.Wrap(err)
	}

	// Setup scheme, override target URL to the destination address
	req.URL.Scheme = "https"
	req.RequestURI = req.URL.Path + "?" + req.URL.RawQuery
	req.URL.Host = sess.teleportCluster.targetAddr
	if sess.teleportCluster.targetAddr == "" {
		req.URL.Host = reversetunnel.LocalKubernetes
	}

	// add origin headers so the service consuming the request on the other site
	// is aware of where it came from
	req.Header.Add("X-Forwarded-Proto", "https")
	req.Header.Add("X-Forwarded-Host", req.Host)
	req.Header.Add("X-Forwarded-Path", req.URL.Path)
	req.Header.Add("X-Forwarded-For", req.RemoteAddr)

	return nil
}

// setupImpersonationHeaders sets up Impersonate-User and Impersonate-Group headers
func setupImpersonationHeaders(log log.FieldLogger, ctx authContext, headers http.Header) error {
	var impersonateUser string
	var impersonateGroups []string
	for header, values := range headers {
		if !strings.HasPrefix(header, "Impersonate-") {
			continue
		}
		switch header {
		case ImpersonateUserHeader:
			if impersonateUser != "" {
				return trace.AccessDenied("%v, user already specified to %q", ImpersonationRequestDeniedMessage, impersonateUser)
			}
			if len(values) == 0 || len(values) > 1 {
				return trace.AccessDenied("%v, invalid user header %q", ImpersonationRequestDeniedMessage, values)
			}
			impersonateUser = values[0]
			if _, ok := ctx.kubeUsers[impersonateUser]; !ok {
				return trace.AccessDenied("%v, user header %q is not allowed in roles", ImpersonationRequestDeniedMessage, impersonateUser)
			}
		case ImpersonateGroupHeader:
			for _, group := range values {
				if _, ok := ctx.kubeGroups[group]; !ok {
					return trace.AccessDenied("%v, group header %q value is not allowed in roles", ImpersonationRequestDeniedMessage, group)
				}
				impersonateGroups = append(impersonateGroups, group)
			}
		default:
			return trace.AccessDenied("%v, unsupported impersonation header %q", ImpersonationRequestDeniedMessage, header)
		}
	}

	impersonateGroups = apiutils.Deduplicate(impersonateGroups)

	// By default, if no kubernetes_users is set (which will be a majority),
	// user will impersonate themselves, which is the backwards-compatible behavior.
	//
	// As long as at least one `kubernetes_users` is set, the forwarder will start
	// limiting the list of users allowed by the client to impersonate.
	//
	// If the users' role set does not include actual user name, it will be rejected,
	// otherwise there will be no way to exclude the user from the list).
	//
	// If the `kubernetes_users` role set includes only one user
	// (quite frequently that's the real intent), teleport will default to it,
	// otherwise it will refuse to select.
	//
	// This will enable the use case when `kubernetes_users` has just one field to
	// link the user identity with the IAM role, for example `IAM#{{external.email}}`
	//
	if impersonateUser == "" {
		switch len(ctx.kubeUsers) {
		// this is currently not possible as kube users have at least one
		// user (user name), but in case if someone breaks it, catch here
		case 0:
			return trace.AccessDenied("assumed at least one user to be present")
		// if there is deterministic choice, make it to improve user experience
		case 1:
			for user := range ctx.kubeUsers {
				impersonateUser = user
				break
			}
		default:
			return trace.AccessDenied(
				"please select a user to impersonate, refusing to select a user due to several kubernetes_users set up for this user")
		}
	}

	if len(impersonateGroups) == 0 {
		for group := range ctx.kubeGroups {
			impersonateGroups = append(impersonateGroups, group)
		}
	}

	if !ctx.teleportCluster.isRemote {
		headers.Set(ImpersonateUserHeader, impersonateUser)

		// Make sure to overwrite the exiting headers, instead of appending to
		// them.
		headers[ImpersonateGroupHeader] = nil
		for _, group := range impersonateGroups {
			headers.Add(ImpersonateGroupHeader, group)
		}
	}
	return nil
}

// catchAll forwards all HTTP requests to the target k8s API server
func (f *Forwarder) catchAll(ctx *authContext, w http.ResponseWriter, req *http.Request) (interface{}, error) {
	sess, err := f.newClusterSession(*ctx)
	if err != nil {
		// This error goes to kubernetes client and is not visible in the logs
		// of the teleport server if not logged here.
		f.log.Errorf("Failed to create cluster session: %v.", err)
		return nil, trace.Wrap(err)
	}
	if err := f.setupForwardingHeaders(sess, req); err != nil {
		// This error goes to kubernetes client and is not visible in the logs
		// of the teleport server if not logged here.
		f.log.Errorf("Failed to set up forwarding headers: %v.", err)
		return nil, trace.Wrap(err)
	}
	rw := newResponseStatusRecorder(w)
	sess.forwarder.ServeHTTP(rw, req)

	if sess.noAuditEvents {
		return nil, nil
	}

	// Emit audit event.
	event := &apievents.KubeRequest{
		Metadata: apievents.Metadata{
			Type: events.KubeRequestEvent,
			Code: events.KubeRequestCode,
		},
		UserMetadata: apievents.UserMetadata{
			User:         ctx.User.GetName(),
			Login:        ctx.User.GetName(),
			Impersonator: ctx.Identity.GetIdentity().Impersonator,
		},
		ConnectionMetadata: apievents.ConnectionMetadata{
			RemoteAddr: req.RemoteAddr,
			LocalAddr:  sess.teleportCluster.targetAddr,
			Protocol:   events.EventProtocolKube,
		},
		ServerMetadata: apievents.ServerMetadata{
			ServerID:        f.cfg.ServerID,
			ServerNamespace: f.cfg.Namespace,
		},
		RequestPath:               req.URL.Path,
		Verb:                      req.Method,
		ResponseCode:              int32(rw.getStatus()),
		KubernetesClusterMetadata: ctx.eventClusterMeta(),
	}
	r := parseResourcePath(req.URL.Path)
	if r.skipEvent {
		return nil, nil
	}
	r.populateEvent(event)
	if err := f.cfg.AuthClient.EmitAuditEvent(f.ctx, event); err != nil {
		f.log.WithError(err).Warn("Failed to emit event.")
	}

	return nil, nil
}

func (f *Forwarder) getExecutor(ctx authContext, sess *clusterSession, req *http.Request) (remotecommand.Executor, error) {
	upgradeRoundTripper := NewSpdyRoundTripperWithDialer(roundTripperConfig{
		ctx:             req.Context(),
		authCtx:         ctx,
		dial:            sess.DialWithContext,
		tlsConfig:       sess.tlsConfig,
		followRedirects: true,
		pingPeriod:      f.cfg.ConnPingPeriod,
	})
	rt := http.RoundTripper(upgradeRoundTripper)
	if sess.creds != nil {
		var err error
		rt, err = sess.creds.wrapTransport(rt)
		if err != nil {
			return nil, trace.Wrap(err)
		}
	}
	return remotecommand.NewSPDYExecutorForTransports(rt, upgradeRoundTripper, req.Method, req.URL)
}

func (f *Forwarder) getDialer(ctx authContext, sess *clusterSession, req *http.Request) (httpstream.Dialer, error) {
	upgradeRoundTripper := NewSpdyRoundTripperWithDialer(roundTripperConfig{
		ctx:             req.Context(),
		authCtx:         ctx,
		dial:            sess.DialWithContext,
		tlsConfig:       sess.tlsConfig,
		followRedirects: true,
		pingPeriod:      f.cfg.ConnPingPeriod,
	})
	rt := http.RoundTripper(upgradeRoundTripper)
	if sess.creds != nil {
		var err error
		rt, err = sess.creds.wrapTransport(rt)
		if err != nil {
			return nil, trace.Wrap(err)
		}
	}
	client := &http.Client{
		Transport: rt,
	}

	return spdy.NewDialer(upgradeRoundTripper, client, req.Method, req.URL), nil
}

// clusterSession contains authenticated user session to the target cluster:
// x509 short lived credentials, forwarding proxies and other data
type clusterSession struct {
	authContext
	parent    *Forwarder
	creds     *kubeCreds
	tlsConfig *tls.Config
	forwarder *forward.Forwarder
	// noAuditEvents is true if this teleport service should leave audit event
	// logging to another service.
	noAuditEvents bool
}

func (s *clusterSession) monitorConn(conn net.Conn, err error) (net.Conn, error) {
	if err != nil {
		return nil, trace.Wrap(err)
	}

	ctx, cancel := context.WithCancel(s.parent.ctx)
	tc, err := srv.NewTrackingReadConn(srv.TrackingReadConnConfig{
		Conn:    conn,
		Clock:   s.parent.cfg.Clock,
		Context: s.parent.cfg.Context,
		Cancel:  cancel,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = srv.StartMonitor(srv.MonitorConfig{
		LockWatcher:           s.parent.cfg.LockWatcher,
		LockTargets:           s.LockTargets(),
		DisconnectExpiredCert: s.disconnectExpiredCert,
		ClientIdleTimeout:     s.clientIdleTimeout,
		Clock:                 s.parent.cfg.Clock,
		Tracker:               tc,
		Conn:                  tc,
		Context:               ctx,
		TeleportUser:          s.User.GetName(),
		ServerID:              s.parent.cfg.ServerID,
		Entry:                 s.parent.log,
		Emitter:               s.parent.cfg.AuthClient,
	})
	if err != nil {
		tc.Close()
		return nil, trace.Wrap(err)
	}
	return tc, nil
}

func (s *clusterSession) Dial(network, addr string) (net.Conn, error) {
	return s.DialWithContext(context.Background(), network, addr)
}

func (s *clusterSession) DialWithContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if s.teleportCluster.targetAddr != "" {
		return s.monitorConn(s.teleportCluster.DialWithContext(ctx, network, addr))
	} else if len(s.teleportClusterEndpoints) > 0 {
		return s.monitorConn(s.dialWithEndpoints(ctx, network, addr))
	}
	return nil, trace.BadParameter("no targets to dial")
}

// This is separated from DialWithEndpoints for testing without monitorConn.
func (s *clusterSession) dialWithEndpoints(ctx context.Context, network, addr string) (net.Conn, error) {
	if len(s.teleportClusterEndpoints) == 0 {
		return nil, trace.BadParameter("no endpoints to dial")
	}

	// Shuffle endpoints to balance load
	shuffledEndpoints := make([]endpoint, len(s.teleportClusterEndpoints))
	copy(shuffledEndpoints, s.teleportClusterEndpoints)
	mathrand.Shuffle(len(shuffledEndpoints), func(i, j int) {
		shuffledEndpoints[i], shuffledEndpoints[j] = shuffledEndpoints[j], shuffledEndpoints[i]
	})

	errs := []error{}
	for _, endpoint := range shuffledEndpoints {
		s.teleportCluster.targetAddr = endpoint.addr
		s.teleportCluster.serverID = endpoint.serverID
		conn, err := s.teleportCluster.DialWithContext(ctx, network, addr)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return conn, nil
	}
	return nil, trace.NewAggregate(errs...)
}

// TODO(awly): unit test this
func (f *Forwarder) newClusterSession(ctx authContext) (*clusterSession, error) {
	if ctx.teleportCluster.isRemote {
		return f.newClusterSessionRemoteCluster(ctx)
	}
	return f.newClusterSessionSameCluster(ctx)
}

func (f *Forwarder) newClusterSessionRemoteCluster(ctx authContext) (*clusterSession, error) {
	sess := &clusterSession{
		parent:      f,
		authContext: ctx,
	}
	var err error
	sess.tlsConfig, err = f.getOrRequestClientCreds(ctx)
	if err != nil {
		f.log.Warningf("Failed to get certificate for %v: %v.", ctx, err)
		return nil, trace.AccessDenied("access denied: failed to authenticate with auth server")
	}
	// remote clusters use special hardcoded URL,
	// and use a special dialer
	sess.teleportCluster.targetAddr = reversetunnel.LocalKubernetes
	transport := f.newTransport(sess.Dial, sess.tlsConfig)

	sess.forwarder, err = forward.New(
		forward.FlushInterval(100*time.Millisecond),
		forward.RoundTripper(transport),
		forward.WebsocketDial(sess.Dial),
		forward.Logger(f.log),
		forward.ErrorHandler(fwdutils.ErrorHandlerFunc(f.formatForwardResponseError)),
	)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return sess, nil
}

func (f *Forwarder) newClusterSessionSameCluster(ctx authContext) (*clusterSession, error) {
	kubeServices, err := f.cfg.CachingAuthClient.GetKubeServices(f.ctx)
	if err != nil && !trace.IsNotFound(err) {
		return nil, trace.Wrap(err)
	}

	if len(kubeServices) == 0 && ctx.kubeCluster == ctx.teleportCluster.name {
		return f.newClusterSessionLocal(ctx)
	}

	// Validate that the requested kube cluster is registered.
	var endpoints []endpoint
outer:
	for _, s := range kubeServices {
		for _, k := range s.GetKubernetesClusters() {
			if k.Name != ctx.kubeCluster {
				continue
			}
			// TODO(awly): check RBAC
			endpoints = append(endpoints, endpoint{
				serverID: fmt.Sprintf("%s.%s", s.GetName(), ctx.teleportCluster.name),
				addr:     s.GetAddr(),
			})
			continue outer
		}
	}
	if len(endpoints) == 0 {
		return nil, trace.NotFound("kubernetes cluster %q is not found in teleport cluster %q", ctx.kubeCluster, ctx.teleportCluster.name)
	}
	// Try to use local credentials first.
	if _, ok := f.creds[ctx.kubeCluster]; ok {
		return f.newClusterSessionLocal(ctx)
	}
	return f.newClusterSessionDirect(ctx, endpoints)
}

func (f *Forwarder) newClusterSessionLocal(ctx authContext) (*clusterSession, error) {
	f.log.Debugf("Handling kubernetes session for %v using local credentials.", ctx)
	sess := &clusterSession{
		parent:      f,
		authContext: ctx,
	}
	if len(f.creds) == 0 {
		return nil, trace.NotFound("this Teleport process is not configured for direct Kubernetes access; you likely need to 'tsh login' into a leaf cluster or 'tsh kube login' into a different kubernetes cluster")
	}
	creds, ok := f.creds[ctx.kubeCluster]
	if !ok {
		return nil, trace.NotFound("kubernetes cluster %q not found", ctx.kubeCluster)
	}
	sess.creds = creds
	sess.authContext.teleportCluster.targetAddr = creds.targetAddr
	sess.tlsConfig = creds.tlsConfig

	// When running inside Kubernetes cluster or using auth/exec providers,
	// kubeconfig provides a transport wrapper that adds a bearer token to
	// requests
	//
	// When forwarding request to a remote cluster, this is not needed
	// as the proxy uses client cert auth to reach out to remote proxy.
	transport, err := creds.wrapTransport(f.newTransport(sess.Dial, sess.tlsConfig))
	if err != nil {
		return nil, trace.Wrap(err)
	}

	sess.forwarder, err = forward.New(
		forward.FlushInterval(100*time.Millisecond),
		forward.RoundTripper(transport),
		forward.WebsocketDial(sess.Dial),
		forward.Logger(f.log),
		forward.ErrorHandler(fwdutils.ErrorHandlerFunc(f.formatForwardResponseError)),
	)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return sess, nil
}

func (f *Forwarder) newClusterSessionDirect(ctx authContext, endpoints []endpoint) (*clusterSession, error) {
	if len(endpoints) == 0 {
		return nil, trace.BadParameter("no kube cluster endpoints provided")
	}

	f.log.WithField("kube_service.endpoints", endpoints).Debugf("Kubernetes session for %v forwarded to remote kubernetes_service instance.", ctx)

	sess := &clusterSession{
		parent:      f,
		authContext: ctx,
		// This session talks to a kubernetes_service, which should handle
		// audit logging. Avoid duplicate logging.
		noAuditEvents: true,
	}
	sess.authContext.teleportClusterEndpoints = endpoints

	var err error
	sess.tlsConfig, err = f.getOrRequestClientCreds(ctx)
	if err != nil {
		f.log.Warningf("Failed to get certificate for %v: %v.", ctx, err)
		return nil, trace.AccessDenied("access denied: failed to authenticate with auth server")
	}

	transport := f.newTransport(sess.Dial, sess.tlsConfig)
	sess.forwarder, err = forward.New(
		forward.FlushInterval(100*time.Millisecond),
		forward.RoundTripper(transport),
		forward.WebsocketDial(sess.Dial),
		forward.Logger(f.log),
		forward.ErrorHandler(fwdutils.ErrorHandlerFunc(f.formatForwardResponseError)),
	)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return sess, nil
}

// DialFunc is a network dialer function that returns a network connection
type DialFunc func(string, string) (net.Conn, error)

func (f *Forwarder) newTransport(dial DialFunc, tlsConfig *tls.Config) *http.Transport {
	return &http.Transport{
		Dial:            dial,
		TLSClientConfig: tlsConfig,
		// Increase the size of the connection pool. This substantially improves the
		// performance of Teleport under load as it reduces the number of TLS
		// handshakes performed.
		MaxIdleConns:        defaults.HTTPMaxIdleConns,
		MaxIdleConnsPerHost: defaults.HTTPMaxIdleConnsPerHost,
		// IdleConnTimeout defines the maximum amount of time before idle connections
		// are closed. Leaving this unset will lead to connections open forever and
		// will cause memory leaks in a long running process.
		IdleConnTimeout: defaults.HTTPIdleTimeout,
	}
}

// getOrCreateRequestContext creates a new certificate request for a given context,
// if there is no active CSR request in progress, or returns an existing one.
// if the new context has been created, cancel function is returned as a
// second argument. Caller should call this function to signal that CSR has been
// completed or failed.
func (f *Forwarder) getOrCreateRequestContext(key string) (context.Context, context.CancelFunc) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ctx, ok := f.activeRequests[key]
	if ok {
		return ctx, nil
	}
	ctx, cancel := context.WithCancel(f.ctx)
	f.activeRequests[key] = ctx
	return ctx, func() {
		cancel()
		f.mu.Lock()
		defer f.mu.Unlock()
		delete(f.activeRequests, key)
	}
}

func (f *Forwarder) getOrRequestClientCreds(ctx authContext) (*tls.Config, error) {
	c := f.getClientCreds(ctx)
	if c == nil {
		return f.serializedRequestClientCreds(ctx)
	}
	return c, nil
}

func (f *Forwarder) getClientCreds(ctx authContext) *tls.Config {
	f.mu.Lock()
	defer f.mu.Unlock()
	creds, ok := f.clientCredentials.Get(ctx.key())
	if !ok {
		return nil
	}
	c := creds.(*tls.Config)
	if !validClientCreds(f.cfg.Clock, c) {
		return nil
	}
	return c
}

func (f *Forwarder) saveClientCreds(ctx authContext, c *tls.Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.clientCredentials.Set(ctx.key(), c, ctx.sessionTTL)
}

func validClientCreds(clock clockwork.Clock, c *tls.Config) bool {
	if len(c.Certificates) == 0 || len(c.Certificates[0].Certificate) == 0 {
		return false
	}
	crt, err := x509.ParseCertificate(c.Certificates[0].Certificate[0])
	if err != nil {
		return false
	}
	// Make sure that the returned cert will be valid for at least 1 more
	// minute.
	return clock.Now().Add(time.Minute).Before(crt.NotAfter)
}

func (f *Forwarder) serializedRequestClientCreds(authContext authContext) (*tls.Config, error) {
	ctx, cancel := f.getOrCreateRequestContext(authContext.key())
	if cancel != nil {
		f.log.Debugf("Requesting new ephemeral user certificate for %v.", authContext)
		defer cancel()
		c, err := f.requestCertificate(authContext)
		if err != nil {
			return nil, trace.Wrap(err)
		}
		return c, f.saveClientCreds(authContext, c)
	}
	// cancel == nil means that another request is in progress, so simply wait until
	// it finishes or fails
	f.log.Debugf("Another request is in progress for %v, waiting until it gets completed.", authContext)
	select {
	case <-ctx.Done():
		c := f.getClientCreds(authContext)
		if c == nil {
			return nil, trace.BadParameter("failed to request ephemeral certificate, try again")
		}
		return c, nil
	case <-f.ctx.Done():
		return nil, trace.BadParameter("forwarder is closing, aborting the request")
	}
}

func (f *Forwarder) requestCertificate(ctx authContext) (*tls.Config, error) {
	f.log.Debugf("Requesting K8s cert for %v.", ctx)
	keyPEM, _, err := f.cfg.Keygen.GenerateKeyPair("")
	if err != nil {
		return nil, trace.Wrap(err)
	}

	privateKey, err := ssh.ParseRawPrivateKey(keyPEM)
	if err != nil {
		return nil, trace.Wrap(err, "failed to parse private key")
	}

	// Note: ctx.UnmappedIdentity can potentially have temporary roles granted via
	// workflow API. Always use the Subject() method to preserve the roles from
	// caller's certificate.
	//
	// Also note: we need to send the UnmappedIdentity which could be a remote
	// user identity. If we used the local mapped identity instead, the
	// receiver of this certificate will think this is a local user and fail to
	// find it in the backend.
	callerIdentity := ctx.UnmappedIdentity.GetIdentity()
	subject, err := callerIdentity.Subject()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	csr := &x509.CertificateRequest{
		Subject: subject,
	}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, csr, privateKey)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})

	response, err := f.cfg.AuthClient.ProcessKubeCSR(auth.KubeCSR{
		Username:    ctx.User.GetName(),
		ClusterName: ctx.teleportCluster.name,
		CSR:         csrPEM,
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	f.log.Debugf("Received valid K8s cert for %v.", ctx)

	cert, err := tls.X509KeyPair(response.Cert, keyPEM)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	pool := x509.NewCertPool()
	for _, certAuthority := range response.CertAuthorities {
		ok := pool.AppendCertsFromPEM(certAuthority)
		if !ok {
			return nil, trace.BadParameter("failed to append certificates, check that kubeconfig has correctly encoded certificate authority data")
		}
	}
	tlsConfig := &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{cert},
	}
	tlsConfig.BuildNameToCertificate()

	return tlsConfig, nil
}

func (f *Forwarder) kubeClusters() []*types.KubernetesCluster {
	var dynLabels map[string]types.CommandLabelV2
	if f.cfg.DynamicLabels != nil {
		dynLabels = types.LabelsToV2(f.cfg.DynamicLabels.Get())
	}

	res := make([]*types.KubernetesCluster, 0, len(f.creds))
	for n := range f.creds {
		res = append(res, &types.KubernetesCluster{
			Name:          n,
			StaticLabels:  f.cfg.StaticLabels,
			DynamicLabels: dynLabels,
		})
	}
	return res
}

type responseStatusRecorder struct {
	http.ResponseWriter
	flusher http.Flusher
	status  int
}

func newResponseStatusRecorder(w http.ResponseWriter) *responseStatusRecorder {
	rec := &responseStatusRecorder{ResponseWriter: w}
	if flusher, ok := w.(http.Flusher); ok {
		rec.flusher = flusher
	}
	return rec
}

func (r *responseStatusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Flush optionally flushes the inner ResponseWriter if it supports that.
// Otherwise, Flush is a noop.
//
// Flush is optionally used by github.com/gravitational/oxy/forward to flush
// pending data on streaming HTTP responses (like streaming pod logs).
//
// Without this, oxy/forward will handle streaming responses by accumulating
// ~32kb of response in a buffer before flushing it.
func (r *responseStatusRecorder) Flush() {
	if r.flusher != nil {
		r.flusher.Flush()
	}
}

func (r *responseStatusRecorder) getStatus() int {
	// http.ResponseWriter implicitly sets StatusOK, if WriteHeader hasn't been
	// explicitly called.
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

package discovery

import (
	"net"

	"github.com/hashicorp/raft"
	"github.com/hashicorp/serf/serf"
	"go.uber.org/zap"
)

/*
Membership is our type wrapping Serf to provide discovery and cluster member-
ship to our service. Users will call New() to create a Membership with the required
configuration and event handler.
*/
type Membership struct {
	Config
	handler Handler
	serf    *serf.Serf
	events  chan serf.Event
	logger  *zap.Logger
}

func New(handler Handler, config Config) (*Membership, error) {
	c := &Membership{
		Config:  config,
		handler: handler,
		logger:  zap.L().Named("membership"),
	}
	if err := c.setupSerf(); err != nil {
		return nil, err
	}
	return c, nil
}

type Config struct {
	NodeName       string
	BindAddr       string
	Tags           map[string]string
	StartJoinAddrs []string
}

// pg 117-118
/*
setupSerf() creates and configures a Serf instance and starts the eventsHandler()
goroutine to handle Serf’s events.
*/
func (m *Membership) setupSerf() (err error) {
	addr, err := net.ResolveTCPAddr("tcp", m.BindAddr)
	if err != nil {
		return err
	}
	config := serf.DefaultConfig()
	config.Init()
	config.MemberlistConfig.BindAddr = addr.IP.String()
	config.MemberlistConfig.BindPort = addr.Port
	m.events = make(chan serf.Event)
	config.EventCh = m.events
	config.Tags = m.Tags
	config.NodeName = m.Config.NodeName
	m.serf, err = serf.Create(config)
	if err != nil {
		return err
	}
	go m.eventHandler()
	if m.StartJoinAddrs != nil {
		_, err = m.serf.Join(m.StartJoinAddrs, true)
		if err != nil {
			return err
		}
	}
	return nil
}

/*
Handler represents some component in our service that needs to know
when a server joins or leaves the cluster.
*/
type Handler interface {
	Join(name, addr string) error
	Leave(name string) error
}

/*
eventHandler() runs in a loop reading events sent by Serf into the events
channel, handling each incoming event according to the event’s type.
*/
func (m *Membership) eventHandler() {
	for e := range m.events {
		switch e.EventType() {
		case serf.EventMemberJoin:
			for _, member := range e.(serf.MemberEvent).Members {
				if m.isLocal(member) {
					continue
				}
				m.handleJoin(member)
			}
		case serf.EventMemberLeave, serf.EventMemberFailed:
			for _, member := range e.(serf.MemberEvent).Members {
				if m.isLocal(member) {
					return
				}
				m.handleLeave(member)
			}
		}
	}
}

/*
When a node joins or leaves the cluster, Serf sends an event to all nodes, including
the node that joined or left the cluster. We check whether the node we got an
event for is the local server so the server doesn’t act on itself—we don’t want
the server to try and replicate itself, for example.
*/

func (m *Membership) handleJoin(member serf.Member) {
	if err := m.handler.Join(
		member.Name,
		member.Tags["rpc_addr"],
	); err != nil {
		m.logError(err, "failed to join", member)
	}
}

func (m *Membership) handleLeave(member serf.Member) {
	if err := m.handler.Leave(
		member.Name,
	); err != nil {
		m.logError(err, "failed to leave", member)
	}
}

/*
isLocal() returns whether the given Serf member is the local member by
checking the members’ names.
*/
func (m *Membership) isLocal(member serf.Member) bool {
	return m.serf.LocalMember().Name == member.Name
}

/*
Members() returns a point-in-time snapshot of the cluster’s Serf members.
*/
func (m *Membership) Members() []serf.Member {
	return m.serf.Members()
}

/*
Leave() tells this member to leave the Serf cluster.
*/
func (m *Membership) Leave() error {
	return m.serf.Leave()
}

/*
logError() logs the given error and message.
*/
func (m *Membership) logError(err error, msg string, member serf.Member) {
	log := m.logger.Error
	if err == raft.ErrNotLeader {
		log = m.logger.Debug
	}
	log(
		msg,
		zap.Error(err),
		zap.String("name", member.Name),
		zap.String("rpc_addr", member.Tags["rpc_addr"]),
	)
}

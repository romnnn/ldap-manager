package base

import (
	"fmt"
	"net"

	gogrpcservice "github.com/romnnn/go-grpc-service"
	"github.com/romnnn/go-grpc-service/auth"
	ldapmanager "github.com/romnnn/ldap-manager"
	ldapconfig "github.com/romnnn/ldap-manager/config"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Rev is set on build time to the git HEAD
var Rev = ""

// LDAPManagerServer ...
type LDAPManagerServer struct {
	gogrpcservice.Service
	Manager       *ldapmanager.LDAPManager
	Authenticator *auth.Authenticator
}

// Shutdown ...
func (s *LDAPManagerServer) Shutdown() {
	s.Service.GracefulStop()
	if s.Manager != nil {
		s.Manager.Close()
	}
}

// NewLDAPManagerServer ...
func NewLDAPManagerServer(ctx *cli.Context) *LDAPManagerServer {
	hasReadonlyUser := ctx.String("openldap-readonly-user") != ""
	baseDN := ctx.String("openldap-base-dn")
	groupsOU := ctx.String("groups-ou")
	usersOU := ctx.String("users-ou")

	groupsDN := ctx.String("groups-dn")
	if groupsDN == "" {
		groupsDN = fmt.Sprintf("ou=%s,%s", groupsOU, baseDN)
	}
	userGroupDN := ctx.String("users-dn")
	if userGroupDN == "" {
		userGroupDN = fmt.Sprintf("ou=%s,%s", usersOU, baseDN)
	}

	manager := &ldapmanager.LDAPManager{
		OpenLDAPConfig: ldapconfig.OpenLDAPConfig{
			Host:                 ctx.String("openldap-host"),
			Port:                 ctx.Int("openldap-port"),
			Protocol:             ctx.String("openldap-protocol"),
			Organization:         ctx.String("openldap-organization"),
			Domain:               ctx.String("openldap-domain"),
			BaseDN:               baseDN,
			AdminPassword:        ctx.String("openldap-admin-password"),
			ConfigPassword:       ctx.String("openldap-config-password"),
			ReadonlyUser:         hasReadonlyUser,
			ReadonlyUserUsername: ctx.String("openldap-readonly-user"),
			ReadonlyUserPassword: ctx.String("openldap-readonly-password"),
			TLS:                  ctx.Bool("openldap-tls"),
			UseRFC2307BISSchema:  ctx.Bool("openldap-use-rfc2307bis"),
		},
		GroupsOU:                 groupsOU,
		UsersOU:                  usersOU,
		GroupsDN:                 groupsDN,
		UserGroupDN:              userGroupDN,
		GroupMembershipAttribute: ctx.String("group-membership-attribute"),
		GroupMembershipUsesUID:   ctx.Bool("group-membership-uses-uid"),
		AccountAttribute:         ctx.String("account-attribute"),
		GroupAttribute:           ctx.String("group-attribute"),
		DefaultUserGroup:         ctx.String("default-user-group"),
		DefaultAdminGroup:        ctx.String("default-admin-group"),
		DefaultUserShell:         ctx.String("default-login-shell"),
		DefaultAdminUsername:     ctx.String("default-admin-username"),
		DefaultAdminPassword:     ctx.String("default-admin-password"),
		ForceCreateAdmin:         ctx.Bool("force-create-admin"),
	}

	return &LDAPManagerServer{
		Service: gogrpcservice.Service{
			Name:               "ldap manager service",
			Version:            ldapmanager.Version,
			BuildTime:          Rev,
			HTTPHealthCheckURL: "/healthz",
		},
		Authenticator: &auth.Authenticator{
			ExpireSeconds: int64(ctx.Int("expire-sec")),
			Issuer:        ctx.String("issuer"),
			Audience:      ctx.String("audience"),
		},
		Manager: manager,
	}
}

// Setup prepares the service
func (s *LDAPManagerServer) Setup(ctx *cli.Context) error {
	// TODO: This is called twice with no reason
	if err := s.Manager.Setup(); err != nil {
		return err
	}
	if err := s.Authenticator.SetupKeys(auth.AuthenticatorKeyConfig{}.Parse(ctx)); err != nil {
		return err
	}
	return nil
}

// Connect starts the service
func (s *LDAPManagerServer) Connect(ctx *cli.Context, listener net.Listener) {
	log.Info("connecting...")
	if err := s.Setup(ctx); err != nil {
		log.Error(err)
		s.Shutdown()
		return
	}
	s.Service.Ready = true
	s.Service.SetHealthy(true)
	log.Infof("%s ready at %s", s.Service.Name, listener.Addr())
}
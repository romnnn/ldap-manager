package ldapmanager

import (
	"crypto/tls"
	"strings"

	"github.com/go-ldap/ldap"
	ldapconfig "github.com/romnnn/ldap-manager/config"
	log "github.com/sirupsen/logrus"
)

// LDAPManager ...
type LDAPManager struct {
	ldapconfig.OpenLDAPConfig

	ldap ldap.Client

	GroupsDN    string
	UserGroupDN string
	// BaseDN      string

	GroupsOU string
	UsersOU  string

	// AdminBindUsername string
	// AdminBindPassword string

	// ReadonlyBindUsername string
	// ReadonlyBindPassword string

	DefaultUserGroup  string
	DefaultAdminGroup string
	DefaultUserShell  string

	GroupMembershipAttribute string
	AccountAttribute         string
	GroupAttribute           string

	GroupMembershipUsesUID bool
	// UseNISSchema           bool
	// RequireStartTLS        bool
}

// NewLDAPManager ...
func NewLDAPManager(cfg ldapconfig.OpenLDAPConfig) LDAPManager {
	return LDAPManager{
		OpenLDAPConfig:           cfg,
		GroupsDN:                 "ou=groups," + cfg.BaseDN,
		UserGroupDN:              "ou=users," + cfg.BaseDN,
		GroupsOU:                 "groups",
		UsersOU:                  "users",
		DefaultUserGroup:         "users",
		DefaultAdminGroup:        "admins",
		DefaultUserShell:         "/bin/bash",
		GroupMembershipAttribute: "uniqueMember", // uniqueMember or memberUID
		AccountAttribute:         "uid",
		GroupAttribute:           "gid",
		GroupMembershipUsesUID:   false,
	}
}

// Close ...
func (m *LDAPManager) Close() {
	if m.ldap != nil {
		m.ldap.Close()
	}
}

// Setup ...
func (m *LDAPManager) Setup() error {
	var err error
	URI := m.OpenLDAPConfig.URI()
	m.ldap, err = ldap.DialURL(URI)
	if err != nil {
		return err
	}

	// Check for TLS
	if strings.HasPrefix(URI, "ldaps:") || m.OpenLDAPConfig.TLS {
		if err := m.ldap.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
			log.Warnf("failed to connect via TLS: %v", err)
			if m.OpenLDAPConfig.TLS {
				return err
			}
		}
	}

	// Bind as the admin user
	if err := m.BindAdmin(); err != nil {
		return err
	}
	if err := m.SetupLDAP(); err != nil {
		return err
	}
	return nil
}

package ldapmanager

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/go-ldap/ldap"
	ldaphash "github.com/romnnn/ldap-manager/hash"
	log "github.com/sirupsen/logrus"
)

// NewAccountRequest ...
type NewAccountRequest struct {
	FirstName, LastName, Username, Password, Email string
	HashingAlgorithm                               ldaphash.LDAPPasswordHashingAlgorithm
}

// Validate ...
func (req *NewAccountRequest) Validate() error {
	if req.Username == "" {
		return errors.New("Must specify username")
	}
	if req.Password == "" {
		return errors.New("Must specify password")
	}
	if req.Email == "" {
		return errors.New("Must specify email")
	}
	if req.FirstName == "" {
		return errors.New("Must specify first name")
	}
	if req.LastName == "" {
		return errors.New("Must specify last name")
	}
	return nil
}

// GetUserListRequest ...
type GetUserListRequest struct {
	ListOptions
	Filters string
	Fields  []string
}

// GetUserList ...
func (m *LDAPManager) GetUserList(req *GetUserListRequest) ([]map[string]string, error) {
	if len(req.Fields) < 1 {
		req.Fields = []string{m.AccountAttribute, "givenname", "sn", "mail"}
	}
	if req.SortKey == "" {
		req.SortKey = m.AccountAttribute
	}
	filter := fmt.Sprintf("(&(%s=*)%s)", m.AccountAttribute, req.Filters)
	result, err := m.ldap.Search(ldap.NewSearchRequest(
		m.UserGroupDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		req.Fields,
		[]ldap.Control{},
	))
	if err != nil {
		return nil, err
	}
	users := make(map[string]map[string]string)
	for _, entry := range result.Entries {
		log.Info(entry)
		if entryKey := entry.GetAttributeValue(req.SortKey); entryKey != "" {
			user := make(map[string]string)
			for _, attr := range entry.Attributes {
				user[attr.Name] = entry.GetAttributeValue(attr.Name)
			}
			users[entryKey] = user
		}
	}
	// Sort for deterministic clipping
	keys := make([]string, 0, len(users))
	for k := range users {
		keys = append(keys, k)
	}
	// Sort
	sort.Slice(keys, func(i, j int) bool {
		asc := keys[i] < keys[j]
		if req.SortOrder == SortDescending {
			return !asc
		}
		return asc
	})
	// Clip
	clippedKeys := keys
	var clippedUsers []map[string]string
	if req.Start >= 0 && req.End < len(keys) && req.Start < req.End {
		clippedKeys = keys[req.Start:req.End]
	}
	for _, key := range clippedKeys {
		clippedUsers = append(clippedUsers, users[key])
	}
	return clippedUsers, nil
}

// AuthenticateUser ...
func (m *LDAPManager) AuthenticateUser(username string, password string) (string, error) {
	// Validate
	if username == "" || password == "" {
		return "", errors.New("must provide username and password")
	}
	// Search for the DN for the given username. If found, try binding with the DN and user's password.
	// If the binding succeeds, return the DN.
	result, err := m.ldap.Search(ldap.NewSearchRequest(
		m.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(%s=%s)", m.AccountAttribute, escape(username)),
		[]string{"dn"},
		[]ldap.Control{},
	))
	if err != nil {
		return "", err
	}
	if len(result.Entries) != 1 {
		return "", fmt.Errorf("zero or multiple (%d) accounts with username %q", len(result.Entries), username)
	}
	// Make sure to always re-bind as admin afterwards
	defer m.BindAdmin()
	userDN := result.Entries[0].DN
	if err := m.ldap.Bind(userDN, password); err != nil {
		return "", fmt.Errorf("unable to bind as %q", username)
	}
	reg, err := regexp.Compile(fmt.Sprintf("%s=(.*?),", m.AccountAttribute))
	if err != nil {
		return "", errors.New("failed to compile regex")
	}
	matchedDN := reg.FindString(userDN)
	return matchedDN, nil
}

// NewAccount ...
func (m *LDAPManager) getNewAccountGroup(username, dn string) (string, int, error) {
	group := m.DefaultUserGroup
	if defaultGID, err := m.GetGroupGID(m.DefaultUserGroup); err == nil {
		return group, defaultGID, nil
	}
	// The default user group might not yet exist
	// Note that a group can only be created with at least one member when using RFC2307BIS
	if err := m.NewGroup(m.DefaultUserGroup, []string{dn}); err != nil {
		// Fall back to create a new group group for the user
		if err := m.NewGroup(username, []string{dn}); err != nil {
			if _, ok := err.(*GroupExistsError); !ok {
				return group, 0, fmt.Errorf("failed to create group for user %q: %v", username, err)
			}
		}
		group = username
	}

	userGroupGID, err := m.GetGroupGID(group)
	if err != nil {
		return group, 0, fmt.Errorf("failed to get GID for group %q: %v", group, err)
	}
	return group, userGroupGID, nil
}

// NewAccount ...
func (m *LDAPManager) NewAccount(req *NewAccountRequest) error {
	// Validate
	if err := req.Validate(); err != nil {
		return err
	}
	// Check for existing user with the same username
	result, err := m.ldap.Search(ldap.NewSearchRequest(
		m.UserGroupDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(%s=%s,%s)", m.AccountAttribute, escape(req.Username), m.UserGroupDN),
		[]string{},
		[]ldap.Control{},
	))
	// fmt.Printf("(%s=%s,%s)\n", m.AccountAttribute, escape(req.Username), m.UserGroupDN)
	if err != nil {
		return fmt.Errorf("failed to check for existing user %q: %v", req.Username, err)
	}
	if len(result.Entries) > 0 {
		return fmt.Errorf("account with username %q already exists", req.Username)
	}
	highestUID, err := m.getHighestID(m.AccountAttribute)
	if err != nil {
		return fmt.Errorf("failed to get highest %s: %v", m.AccountAttribute, err)
	}
	newUID := highestUID + 1
	userDN := fmt.Sprintf("%s=%s,%s", m.AccountAttribute, req.Username, m.UserGroupDN)
	group, GID, err := m.getNewAccountGroup(req.Username, userDN)
	if err != nil {
		return err
	}

	if req.HashingAlgorithm == ldaphash.Default {
		req.HashingAlgorithm = m.HashingAlgorithm
	}

	hashedPassword, err := ldaphash.Password(req.Password, req.HashingAlgorithm)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}
	log.Info(hashedPassword)

	fullName := fmt.Sprintf("%s %s", req.FirstName, req.LastName)
	userAttributes := []ldap.Attribute{
		{Type: "objectClass", Vals: []string{"person", "inetOrgPerson", "posixAccount"}},
		{Type: "uid", Vals: []string{req.Username}},
		{Type: "givenName", Vals: []string{req.FirstName}},
		{Type: "sn", Vals: []string{req.LastName}},
		{Type: "cn", Vals: []string{fullName}},
		{Type: "displayName", Vals: []string{fullName}},
		{Type: "uidNumber", Vals: []string{strconv.Itoa(newUID)}},
		{Type: "gidNumber", Vals: []string{strconv.Itoa(GID)}},
		{Type: "loginShell", Vals: []string{m.DefaultUserShell}},
		{Type: "homeDirectory", Vals: []string{fmt.Sprintf("/home/%s", req.Username)}},
		{Type: "userPassword", Vals: []string{hashedPassword}},
		{Type: "mail", Vals: []string{req.Email}},
	}

	addUserRequest := &ldap.AddRequest{
		DN:         userDN,
		Attributes: userAttributes,
		Controls:   []ldap.Control{},
	}
	log.Debug(addUserRequest)
	if err := m.ldap.Add(addUserRequest); err != nil {
		return fmt.Errorf("failed to add user %q: %v", userDN, err)
	}
	if err := m.AddGroupMember(group, req.Username); err != nil && !isErr(err, ldap.LDAPResultAttributeOrValueExists) {
		return fmt.Errorf("failed to add user %q to group %q: %v", req.Username, group, err)
	}
	if err := m.updateLastID("lastUID", newUID); err != nil {
		return err
	}
	log.Infof("added new account %q (member of group %q)", req.Username, group)
	return nil
}

// DeleteAccount ...
func (m *LDAPManager) DeleteAccount(username string) error {
	if username == "" {
		return errors.New("username must not be empty")
	}
	if err := m.ldap.Del(ldap.NewDelRequest(
		fmt.Sprintf("%s=%s,%s", m.AccountAttribute, escape(username), m.UserGroupDN),
		[]ldap.Control{},
	)); err != nil {
		return err
	}
	log.Infof("removed account %q", username)
	return nil
}

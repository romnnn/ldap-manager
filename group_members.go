package ldapmanager

import (
	"fmt"
	"sort"

	"github.com/go-ldap/ldap"
	log "github.com/sirupsen/logrus"
)

// Group ...
type Group struct {
	Members []string `json:"members" form:"members"`
	Name    string   `json:"name" form:"name"`
	DN      string   `json:"dn" form:"dn"`
}

func (m *LDAPManager) getGroup(groupName string) (*Group, error) {
	result, err := m.ldap.Search(ldap.NewSearchRequest(
		m.GroupsDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(cn=%s)", escapeFilter(groupName)),
		[]string{m.GroupMembershipAttribute},
		[]ldap.Control{},
	))
	if err != nil {
		return nil, err
	}
	if len(result.Entries) != 1 {
		return nil, &ZeroOrMultipleGroupsError{Group: groupName, Count: len(result.Entries)}
	}
	var members []string
	group := result.Entries[0]
	for _, member := range group.GetAttributeValues(m.GroupMembershipAttribute) {
		log.Info(member)
		members = append(members, member)
	}
	return &Group{
		Members: members,
		Name:    groupName,
		DN:      group.DN,
	}, nil
}

// IsGroupMember ...
func (m *LDAPManager) IsGroupMember(username, groupName string) (bool, error) {
	result, err := m.findGroup(groupName, []string{"dn", m.GroupMembershipAttribute})
	if err != nil {
		return false, err
	}
	if len(result.Entries) != 1 {
		return false, &ZeroOrMultipleGroupsError{Group: groupName, Count: len(result.Entries)}
	}
	if !m.GroupMembershipUsesUID {
		// "${LDAP['account_attribute']}=$username,${LDAP['user_dn']}";
		username = fmt.Sprintf("%s=%s,%s", m.AccountAttribute, username, m.UserGroupDN)
	}
	// preg_grep ("/^${username}$/i", $result[0][$LDAP['group_membership_attribute']])
	for _, member := range result.Entries[0].GetAttributeValues(m.GroupMembershipAttribute) { // uniqueMember or memberUID
		if member == username {
			return true, nil
		}
	}
	return false, nil
}

// GetGroup ...
func (m *LDAPManager) GetGroup(groupName string, options *ListOptions) (*Group, error) {
	/*result, err := m.ldap.Search(ldap.NewSearchRequest(
		m.GroupsDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(cn=%s)", escapeFilter(groupName)),
		[]string{m.GroupMembershipAttribute},
		[]ldap.Control{},
	))
	if err != nil {
		return nil, err
	}
	if len(result.Entries) != 1 {
		return nil, &ZeroOrMultipleGroupsError{Group: groupName, Count: len(result.Entries)}
	}
	var members []string
	group := result.Entries[0]
	for _, member := range group.GetAttributeValues(m.GroupMembershipAttribute) {
		log.Info(member)
		// TODO
			reg, err := regexp.Compile(fmt.Sprintf("%s=(.*?),", m.AccountAttribute))
			if err != nil {
				return "", errors.New("failed to compile regex")
			}
			matchedDN := reg.FindString(userDN)


		// if member.Key != "count" && member.Value != "" {
		// $this_member = preg_replace("/^.*?=(.*?),.*", "$1", $value);
		// array_push($records, $this_member);
		// }
	}
	*/
	group, err := m.getGroup(groupName)
	if err != nil {
		return nil, err
	}
	normGroup := &Group{Name: group.Name}

	// Convert member DN's to usernames
	for _, memberDN := range group.Members {
		if memberUsername, err := extractAttribute(memberDN, m.AccountAttribute); err == nil && memberUsername != "" {
			normGroup.Members = append(normGroup.Members, memberUsername)
		}
	}

	// Sort
	sort.Slice(normGroup.Members, func(i, j int) bool {
		asc := normGroup.Members[i] < normGroup.Members[j]
		if options.SortOrder == SortDescending {
			return !asc
		}
		return asc
	})
	// Clip
	if options.Start >= 0 && options.End < len(normGroup.Members) && options.Start < options.End {
		normGroup.Members = normGroup.Members[options.Start:options.End]
		return normGroup, nil
	}
	return normGroup, nil
}

// AddGroupMember ...
func (m *LDAPManager) AddGroupMember(groupName string, username string) error {
	groupDN := fmt.Sprintf("cn=%s,%s", escapeDN(groupName), m.GroupsDN)
	username = escapeDN(username)
	if !m.GroupMembershipUsesUID {
		username = fmt.Sprintf("%s=%s,%s", m.AccountAttribute, username, m.UserGroupDN)
	}

	modifyRequest := ldap.NewModifyRequest(
		groupDN,
		[]ldap.Control{},
	)
	modifyRequest.Add(m.GroupMembershipAttribute, []string{username})
	log.Debug(modifyRequest)
	if err := m.ldap.Modify(modifyRequest); err != nil {
		return err
	}
	log.Infof("added user %q to group %q", username, groupName)
	return nil
}

// DeleteGroupMember ...
func (m *LDAPManager) DeleteGroupMember(groupName string, username string) error {
	groupDN := fmt.Sprintf("cn=%s,%s", escapeDN(groupName), m.GroupsDN)
	if !m.GroupMembershipUsesUID {
		username = fmt.Sprintf("%s=%s,%s", m.AccountAttribute, username, m.UserGroupDN)
	}

	modifyRequest := ldap.NewModifyRequest(
		groupDN,
		[]ldap.Control{},
	)
	modifyRequest.Delete(m.GroupMembershipAttribute, []string{username})
	log.Debug(modifyRequest)
	if err := m.ldap.Modify(modifyRequest); err != nil {
		return err
	}
	log.Infof("removed user %q from group %q", username, groupName)
	return nil
}

package ldapmanager

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/go-ldap/ldap"
	log "github.com/sirupsen/logrus"
)

// ZeroOrMultipleGroupsError ...
type ZeroOrMultipleGroupsError struct {
	Group string
	Count int
}

// Status ...
func (e *ZeroOrMultipleGroupsError) Status() int {
	if e.Count > 1 {
		return http.StatusConflict
	}
	return http.StatusNotFound
}

// Error ...
func (e *ZeroOrMultipleGroupsError) Error() string {
	if e.Count > 1 {
		return fmt.Sprintf("multiple (%d) groups with name %q", e.Count, e.Group)
	}
	return fmt.Sprintf("no group with name %q", e.Group)
}

// IsGroupMember ...
func (m *LDAPManager) IsGroupMember(username, groupName string) (bool, error) {
	result, err := m.findGroup(groupName, []string{"dn", m.GroupMembershipAttribute})
	if err != nil {
		return false, err
	}
	if len(result.Entries) != 1 {
		return false, fmt.Errorf("user %q does not exist or too many entries returned", username)
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

// GetGroupMembers ...
func (m *LDAPManager) GetGroupMembers(groupName string, start, end int, sortOrder string) ([]string, error) {
	result, err := m.ldap.Search(ldap.NewSearchRequest(
		m.GroupsDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(cn=%s)", escape(groupName)),
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
		/*
			reg, err := regexp.Compile(fmt.Sprintf("%s=(.*?),", m.AccountAttribute))
			if err != nil {
				return "", errors.New("failed to compile regex")
			}
			matchedDN := reg.FindString(userDN)
		*/

		// if member.Key != "count" && member.Value != "" {
		// $this_member = preg_replace("/^.*?=(.*?),.*/", "$1", $value);
		// array_push($records, $this_member);
		// }
	}

	// Sort
	sort.Slice(members, func(i, j int) bool {
		asc := members[i] < members[j]
		if sortOrder == SortDescending {
			return !asc
		}
		return asc
	})
	// Clip
	if start >= 0 && end < len(members) && start < end {
		return members[start:end], nil
	}
	return members, nil
}

// AddGroupMember ...
func (m *LDAPManager) AddGroupMember(groupName string, username string) error {
	groupDN := fmt.Sprintf("cn=%s,%s", escape(groupName), m.GroupsDN)
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
	groupDN := fmt.Sprintf("cn=%s,%s", escape(groupName), m.GroupsDN)
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

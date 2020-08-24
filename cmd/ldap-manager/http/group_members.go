package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/neko-neko/echo-logrus/v2/log"
	ldapmanager "github.com/romnnn/ldap-manager"
	pb "github.com/romnnn/ldap-manager/grpc/ldap-manager"
)

type groupMemberRequest struct {
	Username string `json:"username" xml:"username"`
}

func (s *LDAPManagerServer) addGroupMemberHandler(c echo.Context) error {
	group := c.Param("group")
	var req groupMemberRequest
	if err := c.Bind(&req); err != nil {
		log.Error(err)
		return err
	}
	allowNonExistent := false
	if err := s.Manager.AddGroupMember(&pb.GroupMember{Group: group, Username: req.Username}, allowNonExistent); err != nil {
		switch err.(type) {
		case *ldapmanager.GroupValidationError:
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		case *ldapmanager.ZeroOrMultipleGroupsError:
			return echo.NewHTTPError(err.(*ldapmanager.ZeroOrMultipleGroupsError).Status(), err.Error())
		}
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to add group member")
	}
	return c.JSONPretty(http.StatusOK, nil, "  ")
}

func (s *LDAPManagerServer) removeGroupMemberHandler(c echo.Context) error {
	group := c.Param("group")
	var req groupMemberRequest
	if err := c.Bind(&req); err != nil {
		log.Error(err)
		return err
	}
	allowDeleteOfDefaultGroups := false
	if err := s.Manager.DeleteGroupMember(&pb.GroupMember{Group: group, Username: req.Username}, allowDeleteOfDefaultGroups); err != nil {
		switch err.(type) {
		case *ldapmanager.GroupValidationError:
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		case *ldapmanager.ZeroOrMultipleGroupsError:
			return echo.NewHTTPError(err.(*ldapmanager.ZeroOrMultipleGroupsError).Status(), err.Error())
		}
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete group member")
	}
	return c.JSONPretty(http.StatusOK, nil, "  ")
}

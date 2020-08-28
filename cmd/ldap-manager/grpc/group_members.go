package grpc

import (
	"context"

	ldapmanager "github.com/romnnn/ldap-manager"
	pb "github.com/romnnn/ldap-manager/grpc/ldap-manager"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsGroupMember ...
func (s *LDAPManagerServer) IsGroupMember(ctx context.Context, in *pb.IsGroupMemberRequest) (*pb.GroupMemberStatus, error) {
	memberStatus, err := s.Manager.IsGroupMember(in)
	if err != nil {
		if appErr, safe := err.(ldapmanager.Error); safe {
			return &pb.GroupMemberStatus{}, toStatus(appErr)
		}
		log.Error(err)
		return &pb.GroupMemberStatus{}, status.Error(codes.Internal, "error while checking if user is member")
	}
	return memberStatus, nil
}

// GetGroup ...
func (s *LDAPManagerServer) GetGroup(ctx context.Context, in *pb.GetGroupRequest) (*pb.Group, error) {
	group, err := s.Manager.GetGroup(in)
	if err != nil {
		if appErr, safe := err.(ldapmanager.Error); safe {
			return &pb.Group{}, toStatus(appErr)
		}
		log.Error(err)
		return &pb.Group{}, status.Error(codes.Internal, "error while getting group")
	}
	return group, nil
}

// GetUserGroups ...
func (s *LDAPManagerServer) GetUserGroups(ctx context.Context, in *pb.GetUserGroupsRequest) (*pb.GroupList, error) {
	groups, err := s.Manager.GetUserGroups(in)
	if err != nil {
		if appErr, safe := err.(ldapmanager.Error); safe {
			return &pb.GroupList{}, toStatus(appErr)
		}
		log.Error(err)
		return &pb.GroupList{}, status.Error(codes.Internal, "error while getting groups")
	}
	return groups, nil
}

// AddGroupMember ...
func (s *LDAPManagerServer) AddGroupMember(ctx context.Context, in *pb.GroupMember) (*pb.Empty, error) {
	if err := s.Manager.AddGroupMember(in, false); err != nil {
		if appErr, safe := err.(ldapmanager.Error); safe {
			return &pb.Empty{}, toStatus(appErr)
		}
		log.Error(err)
		return &pb.Empty{}, status.Error(codes.Internal, "error while adding group member")
	}
	return &pb.Empty{}, nil
}

// DeleteGroupMember ...
func (s *LDAPManagerServer) DeleteGroupMember(ctx context.Context, in *pb.GroupMember) (*pb.Empty, error) {
	if err := s.Manager.DeleteGroupMember(in, false); err != nil {
		if appErr, safe := err.(ldapmanager.Error); safe {
			return &pb.Empty{}, toStatus(appErr)
		}
		log.Error(err)
		return &pb.Empty{}, status.Error(codes.Internal, "error while deleting group member")
	}
	return &pb.Empty{}, nil
}

package main

import (
	"context"
	"log"
	"net"
	"sync"

	pb "backend-boilerplate/pb"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type server struct {
	pb.UnimplementedUserServiceServer
	mu    sync.RWMutex
	users map[string]*pb.User
}

func newServer() *server {
	return &server{
		users: make(map[string]*pb.User),
	}
}

func (s *server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	if req.Email == "" || req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "email and name are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate email
	for _, u := range s.users {
		if u.Email == req.Email {
			return nil, status.Error(codes.AlreadyExists, "email already exists")
		}
	}

	now := timestamppb.Now()
	user := &pb.User{
		Id:        uuid.New().String(),
		Email:     req.Email,
		Name:      req.Name,
		Status:    pb.UserStatus_USER_STATUS_ACTIVE,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.users[user.Id] = user
	return user, nil
}

func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return user, nil
}

func (s *server) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	if req.Email != nil {
		// Check for duplicate email
		for _, u := range s.users {
			if u.Id != req.Id && u.Email == *req.Email {
				return nil, status.Error(codes.AlreadyExists, "email already exists")
			}
		}
		user.Email = *req.Email
	}

	if req.Name != nil {
		user.Name = *req.Name
	}

	if req.Status != nil {
		user.Status = *req.Status
	}

	user.UpdatedAt = timestamppb.Now()
	return user, nil
}

func (s *server) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[req.Id]; !exists {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	delete(s.users, req.Id)
	return &pb.DeleteUserResponse{Success: true}, nil
}

func (s *server) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 {
		req.PerPage = 10
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var users []*pb.User
	for _, user := range s.users {
		users = append(users, user)
	}

	// Calculate pagination
	start := (int(req.Page) - 1) * int(req.PerPage)
	end := start + int(req.PerPage)
	total := len(users)

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	return &pb.ListUsersResponse{
		Users:   users[start:end],
		Total:   int32(total),
		Page:    req.Page,
		PerPage: req.PerPage,
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, newServer())

	reflection.Register(s)

	log.Printf("Server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

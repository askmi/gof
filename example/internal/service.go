package internal

import (
	"context"
	"errors"
)

type Service struct {
	s *Store
}

func NewService(s *Store) Service {
	return Service{s: s}
}

func (s *Service) AddUser(ctx context.Context, u User) (User, error) {
	if u.ID != 0 {
		return ZeroUser, errors.New("Add: User.ID is not 0")
	}
	v := s.s.Create(ctx, u)
	return v, nil
}

func (s *Service) EditUser(ctx context.Context, u User) (User, error) {
	if u.ID != 0 {
		return ZeroUser, errors.New("Edit: User.ID is 0")
	}
	if !s.s.Exists(ctx, u) {
		return ZeroUser, errors.New("Edit: User not found")
	}
	v := s.s.Update(ctx, u)
	return v, nil
}

func (s *Service) DeleteUser(ctx context.Context, ID int) error {
	s.s.Remove(ctx, ID)
	return nil
}

func (s *Service) GetUser(ctx context.Context, ID int) (User, error) {
	v, ok := s.s.FindByID(ctx, ID)
	if !ok {
		return ZeroUser, errors.New("Get: User not found")
	}
	return v, nil
}

func (s *Service) Search(ctx context.Context, q any) ([]User, error) {
	res := s.s.All(ctx, 0, s.s.Count(ctx))
	return res, nil
}

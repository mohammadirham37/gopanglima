// Package services contains the example's application services.
package services

import (
	"context"
	"errors"
	"sort"
	"sync"
)

// ErrUserNotFound is returned when a user does not exist.
var ErrUserNotFound = errors.New("user not found")

// User is a user record.
type User struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UserService manages users.
type UserService interface {
	List(ctx context.Context) ([]User, error)
	Find(ctx context.Context, id int64) (User, error)
	Create(ctx context.Context, name, email string) (User, error)
	Delete(ctx context.Context, id int64) error
}

// MemoryUserService is an in-memory, development-only UserService. Data is
// lost when the process exits.
type MemoryUserService struct {
	mu     sync.RWMutex
	nextID int64
	users  map[int64]User
}

// NewMemoryUserService returns an empty MemoryUserService.
func NewMemoryUserService() *MemoryUserService {
	return &MemoryUserService{nextID: 1, users: make(map[int64]User)}
}

// List returns all users ordered by ID.
func (s *MemoryUserService) List(ctx context.Context) ([]User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Find returns the user with the given ID.
func (s *MemoryUserService) Find(ctx context.Context, id int64) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

// Create stores a new user.
func (s *MemoryUserService) Create(ctx context.Context, name, email string) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	u := User{ID: s.nextID, Name: name, Email: email}
	s.users[u.ID] = u
	s.nextID++
	return u, nil
}

// Delete removes the user with the given ID.
func (s *MemoryUserService) Delete(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[id]; !ok {
		return ErrUserNotFound
	}
	delete(s.users, id)
	return nil
}

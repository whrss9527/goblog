package filestore

import (
	"fmt"

	"goblog/internal/pkg/model"
)

func (r *FileRepository) GetUserByEmail(email string) (model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return model.User{}, fmt.Errorf("user not found: %s", email)
}

func (r *FileRepository) AddUser(user model.User) (uint, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var maxId uint
	for _, u := range r.users {
		if u.Id > maxId {
			maxId = u.Id
		}
	}
	user.Id = maxId + 1
	r.users = append(r.users, user)
	if err := r.saveJSON("users.json", r.users); err != nil {
		return 0, err
	}
	return user.Id, nil
}

func (r *FileRepository) UpdateUser(user model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, u := range r.users {
		if u.Id == user.Id {
			r.users[i].Password = user.Password
			return r.saveJSON("users.json", r.users)
		}
	}
	return fmt.Errorf("user not found: %d", user.Id)
}

func (r *FileRepository) DeleteUserByEmail(email string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, u := range r.users {
		if u.Email == email {
			r.users = append(r.users[:i], r.users[i+1:]...)
			return r.saveJSON("users.json", r.users)
		}
	}
	return fmt.Errorf("user not found: %s", email)
}

func (r *FileRepository) GetAllUsers() ([]model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]model.User, len(r.users))
	copy(result, r.users)
	return result, nil
}

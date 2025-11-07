package main

import (
	"fmt"
)

func main() {
	fmt.Println(MySQLUserSearcher{}.SearchUser())
}

type User struct {
	id   int
	name string
}

type UserSearcher interface {
	SearchUser() []User
}

type MySQLUserSearcher struct {
	name string
}

func (userSearcher MySQLUserSearcher) SearchUser() []User {
	return []User{
		{id: 10, name: userSearcher.name},
	}
}

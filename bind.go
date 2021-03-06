package ldapc

import (
	"errors"
	"fmt"

	"gopkg.in/ldap.v2"
)

// Bind provide to auth method
type Bind interface {
	auth(conn *ldap.Conn, username, password string) (*ldap.Entry, error)
}

func userFilter(filter string, username string) string {
	return fmt.Sprintf(filter, username)
}

func search(conn *ldap.Conn, username string, baseDN string, filter string) (*ldap.Entry, error) {
	request := ldap.NewSearchRequest(baseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false, filter, nil, nil)

	result, err := conn.Search(request)
	if err != nil {
		return nil, fmt.Errorf("LDAP Search failed! (%v)", err)
	} else if len(result.Entries) < 1 {
		return nil, fmt.Errorf("Failed search using filter: %v", filter)
	} else if len(result.Entries) > 1 {
		return nil, fmt.Errorf("Filter '%s' returned more than one user.", filter)
	}

	return result.Entries[0], nil
}

// AuthBind struct required to run ldap operation.
type AuthBind struct {
	// 1. LDAP Bind operation with BindDN and BindPassword. Bind success then step 2.
	// 2. LDAP Search operation with user's name, BaseDN and Filter. Search success then get UserDN and user's entry
	// 3. LDAP Bind operation with UserDN and user's password.
	// 4. Finish
	BindDN       string // for LDAP Server authentication.
	BindPassword string // BindDN password
	BaseDN       string // Base search path for users
	Filter       string // An LDAP filter declaring when a user should be allowed to log in. The `%s` matching parameter will be substituted with the user's username.
}

func (b *AuthBind) auth(conn *ldap.Conn, username, password string) (*ldap.Entry, error) {
	err := conn.Bind(b.BindDN, b.BindPassword)
	if err != nil {
		return nil, fmt.Errorf("LDAP Bind error, %s:%v", b.BindDN, err)
	}

	entry, err := search(conn, username, b.BaseDN, userFilter(b.Filter, username))
	if err != nil {
		return nil, err
	}

	userDN := entry.DN
	if len(userDN) == 0 {
		return nil, errors.New("LDAP search was successful, but found no DN!")
	}

	err = conn.Bind(userDN, password)
	if err != nil {
		return nil, fmt.Errorf("LDAP Bind failed. user dn: %v", userDN)
	}

	return entry, nil

}

// DirectBind required to run ldap operation.
type DirectBind struct {
	// 1. LDAP Bind operation with UserDN and user's password. Bind success then step 2.
	// 2. LDAP Search operation with user's name, UserDN and Filter. Search success then get user's entry
	// 3. Finish
	UserDN string // A template to use as the user's DN. The `%s` matching parameter will be substituted with the user's username.
	Filter string // An LDAP filter declaring when a user should be allowed to log in. The `%s` matching parameter will be substituted with the user's username.
}

func (b *DirectBind) auth(conn *ldap.Conn, username, password string) (*ldap.Entry, error) {
	userDN := fmt.Sprintf(b.UserDN, username)

	err := conn.Bind(userDN, password)
	if err != nil {
		return nil, fmt.Errorf("LDAP Bind error, %s:%v", userDN, err)
	}

	entry, err := search(conn, username, userDN, userFilter(b.Filter, username))
	if err != nil {
		return nil, err
	}

	return entry, nil
}

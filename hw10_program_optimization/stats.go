package hw10programoptimization

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/mailru/easyjson"
)

type User struct {
	Email string
}

type DomainStat map[string]int

func GetDomainStat(r io.Reader, domain string) (DomainStat, error) {
	domain = fmt.Sprintf(".%s", domain)
	domainStat := make(DomainStat)
	buf := bufio.NewReader(r)

	var user *User
	var err error
	for {
		user, err = getUser(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		updateDomainStat(domainStat, user, domain)
	}

	if user != nil {
		updateDomainStat(domainStat, user, domain)
	}

	return domainStat, nil
}

func updateDomainStat(domainStat DomainStat, user *User, domain string) {
	if strings.HasSuffix(user.Email, domain) {
		foundDomain := strings.ToLower(strings.SplitN(user.Email, "@", 2)[1])
		num := domainStat[foundDomain]
		num++
		domainStat[foundDomain] = num
	}
}

func getUser(buf *bufio.Reader) (*User, error) {
	data, err := buf.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}

	user := &User{}
	if parseErr := easyjson.Unmarshal(data, user); parseErr != nil {
		return nil, parseErr
	}

	return user, err
}

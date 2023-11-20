package jmp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmihailenco/msgpack/v5"
)

func TestDecode(t *testing.T) {
	type User struct {
		ID   int64
		Name string
		ABCD string
	}
	user := User{ID: 1, Name: "jimmy", ABCD: "abcd"}
	d, err := msgpack.Marshal(&user)
	if err != nil {
		t.Fatal(err)
	}
	var userDecoded User
	Decode(d, &userDecoded)
	assert.Equal(t, userDecoded.ID, user.ID)
	assert.Equal(t, userDecoded.ID, user.ID)
	assert.Equal(t, userDecoded.ABCD, user.ABCD)
}

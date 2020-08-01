package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	jwt "github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/sha3"
)

type User struct {
	No       uint64    `json:"no" db:"no"`
	Id       string    `json:"id" db:"id"`
	Nickname string    `json:"nickname" db:"nickname"`
	CreateAt time.Time `json:"created_at" db:"created_at"`
}

func (user *User) Get() {
	database.NewSession(nil).Select("*").From("users").Where("no=?", user.No).Load(user)
}

func GetUsers(w rest.ResponseWriter, r *rest.Request) {
	no, _ := strconv.ParseUint(r.PathParam("no"), 10, 64)
	user := User{No: uint64(no)}
	user.Get()
	w.WriteJson(user)
}

func PostUsers(w rest.ResponseWriter, r *rest.Request) {
	props := map[string]interface{}{}
	r.DecodeJsonPayload(&props)

	session := database.NewSession(nil)
	hashed := sha3.Sum256([]byte(props["password"].(string)))
	res, err := session.InsertInto("users").Columns("id", "password", "nickname").Values(props["id"], hashed[:], props["nickname"]).Exec()
	if err != nil {
		w.WriteHeader(http.StatusConflict)
		w.WriteJson(map[string]interface{}{
			"error": "conflict",
		})
		return
	}

	no, _ := res.LastInsertId()
	user := User{No: uint64(no)}
	user.Get()
	w.WriteJson(struct {
		User  User   `json:"user"`
		Token string `json:"token"`
	}{
		User:  user,
		Token: newToken(user.No),
	})
}

func PostToken(w rest.ResponseWriter, r *rest.Request) {
	props := map[string]interface{}{}
	r.DecodeJsonPayload(&props)

	fmt.Println(props)

	hashed := sha3.Sum256([]byte(props["password"].(string)))
	user := User{}
	rows, _ := database.NewSession(nil).Select("*").From("users").Where("id=? AND `password`=?", props["id"], hashed[:]).Load(&user)

	if rows == 0 {
		w.WriteHeader(http.StatusNotFound)
		w.WriteJson(map[string]interface{}{
			"error": "notfound",
		})
		return
	}

	w.WriteJson(map[string]interface{}{
		"token": newToken(user.No),
	})
}

func newToken(no uint64) string {
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"no": no,
	}).SignedString([]byte(configs.TokenSecret))

	return token
}

type TokenClaims struct {
	No uint64 `json:"no"`
	jwt.StandardClaims
}

func parseToken(tokenString string) (uint64, bool) {
	token, _ := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(configs.TokenSecret), nil
	})

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims.No, true
	}

	return 0, false
}

func writeAuthError(w rest.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
	w.WriteJson(map[string]interface{}{
		"error": "unauthroized",
	})
}
